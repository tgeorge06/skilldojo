package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrateIsIdempotent(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	n, err := Migrate(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("applied %d migrations, want >= 1", n)
	}
	again, err := Migrate(ctx, d)
	if err != nil || again != 0 {
		t.Fatalf("second migrate applied %d, err %v", again, err)
	}
	var fk int
	if err := d.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d, err %v; want 1", fk, err)
	}
	var mode string
	if err := d.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, err %v; want wal", mode, err)
	}
}
