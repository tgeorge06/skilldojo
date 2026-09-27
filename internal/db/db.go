// Package db opens the SQLite database and applies embedded migrations at
// boot. One process owns the file; WAL mode lets reads overlap the single
// writer and busy_timeout absorbs short contention.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open opens (creating if needed) the database at path with the pragmas the
// app relies on. Pass ":memory:" for tests.
func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)"
	if path == ":memory:" {
		// A shared in-memory DB so every pooled connection sees the same data.
		dsn = "file:memdb?mode=memory&cache=shared&_pragma=foreign_keys(ON)"
	}
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", path, err)
	}
	d.SetMaxOpenConns(4)
	d.SetConnMaxLifetime(0)
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("db: ping %s: %w", path, err)
	}
	return d, nil
}

// Migrate applies every embedded migration newer than the recorded schema
// version, each in its own transaction. Files are named NNNN_description.sql.
func Migrate(ctx context.Context, d *sql.DB) (applied int, err error) {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return 0, fmt.Errorf("db: create schema_version: %w", err)
	}
	var current int
	if err := d.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current); err != nil {
		return 0, fmt.Errorf("db: read schema_version: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return 0, err
	}
	type migration struct {
		version int
		name    string
	}
	var pending []migration
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(name, "_")
		v, convErr := strconv.Atoi(prefix)
		if !ok || convErr != nil || v <= 0 {
			return 0, fmt.Errorf("db: migration %q is not named NNNN_description.sql", name)
		}
		if v > current {
			pending = append(pending, migration{v, name})
		}
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	for _, m := range pending {
		body, err := migrationFS.ReadFile("migrations/" + m.name)
		if err != nil {
			return applied, err
		}
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return applied, err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return applied, fmt.Errorf("db: apply %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version (version, applied_at) VALUES (?, ?)`,
			m.version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			tx.Rollback()
			return applied, fmt.Errorf("db: record %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}
