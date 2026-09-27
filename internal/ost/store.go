package ost

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound   = errors.New("ost: attempt not found")
	ErrBadRequest = errors.New("ost: bad request")
	ErrFinished   = errors.New("ost: this test has already been turned in")
)

const (
	SubjectMath = "math"
	MinGrade    = 3
	MaxGrade    = 5
	tsLayout    = "2006-01-02T15:04:05.000000000Z"
)

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

// Child identifies whose attempt it is. Every query carries both ids.
type Child struct {
	AccountID int64
	ChildID   int64
	Grade     int
}

// Store keeps attempts.
type Store struct{ db *sql.DB }

// New wraps the database.
func New(db *sql.DB) *Store { return &Store{db: db} }

// Attempt is one practice test, as sent to the child (answers held server-side).
type Attempt struct {
	ID         string            `json:"id"`
	Subject    string            `json:"subject"`
	Grade      int               `json:"grade"`
	Items      []PublicItem      `json:"items"`
	Answers    map[string]Answer `json:"answers"`
	StartedAt  string            `json:"started_at"`
	FinishedAt string            `json:"finished_at,omitempty"`
	Report     *Report           `json:"report,omitempty"`
}

// ItemResult is one graded item for the review screen.
type ItemResult struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Standard    string   `json:"standard"`
	DOK         int      `json:"dok"`
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Choices     []string `json:"choices,omitempty"`
	Answer      []int    `json:"answer,omitempty"`
	Numeric     string   `json:"numeric,omitempty"`
	Given       Answer   `json:"given"`
	Answered    bool     `json:"answered"`
	Right       bool     `json:"right"`
	Explanation string   `json:"explanation"`
}

// Tally is right/total for one category or standard.
type Tally struct {
	Name    string `json:"name"`
	Right   int    `json:"right"`
	Total   int    `json:"total"`
	Percent int    `json:"percent"`
}

// Report is the graded result of one attempt.
type Report struct {
	Score      int          `json:"score"`
	Total      int          `json:"total"`
	Percent    int          `json:"percent"`
	Level      string       `json:"level"`
	Categories []Tally      `json:"categories"`
	Standards  []Tally      `json:"standards"`
	Items      []ItemResult `json:"items"`
}

// Summary is one row of a child's attempt history.
type Summary struct {
	ID         string
	Grade      int
	StartedAt  time.Time
	FinishedAt time.Time
	Finished   bool
	Answered   int
	Score      int
	Total      int
	Percent    int
	Level      string
	Categories []Tally
	Standards  []Tally
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func newSeed() (uint64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

// ValidGrade clamps a child's grade onto the tested range.
func ValidGrade(grade int) bool { return grade >= MinGrade && grade <= MaxGrade }

// NearestGrade picks the test a child of any grade should start with.
func NearestGrade(grade int) int {
	return min(max(grade, MinGrade), MaxGrade)
}

// Start resumes the child's unfinished attempt for the grade, or creates
// a fresh one. Attempts are unlimited.
func (s *Store) Start(ctx context.Context, child Child, grade int, now time.Time) (Attempt, error) {
	if !ValidGrade(grade) {
		return Attempt{}, fmt.Errorf("%w: practice tests cover grades %d to %d", ErrBadRequest, MinGrade, MaxGrade)
	}
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM ost_attempts WHERE child_id = ? AND account_id = ? AND subject = ? AND grade = ? AND finished_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`, child.ChildID, child.AccountID, SubjectMath, grade).Scan(&id)
	if err == nil {
		return s.Attempt(ctx, child, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Attempt{}, err
	}
	seed, err := newSeed()
	if err != nil {
		return Attempt{}, err
	}
	items, err := Build(grade, seed)
	if err != nil {
		return Attempt{}, err
	}
	id, err = newID()
	if err != nil {
		return Attempt{}, err
	}
	itemsJSON, _ := json.Marshal(items)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO ost_attempts (id, account_id, child_id, subject, grade, seed, items_json, started_at, updated_at, total)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, child.AccountID, child.ChildID, SubjectMath, grade, int64(seed), string(itemsJSON), ts(now), ts(now), len(items))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			// Another request started this child's test first; resume it.
			return s.Start(ctx, child, grade, now)
		}
		return Attempt{}, err
	}
	return s.Attempt(ctx, child, id)
}

type row struct {
	id, subject     string
	grade           int
	items           []Item
	answers         map[string]Answer
	started, finish sql.NullString
	report          string
}

func (s *Store) load(ctx context.Context, child Child, id string) (row, error) {
	var rw row
	var itemsJSON, answersJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, subject, grade, items_json, answers_json, started_at, finished_at, report_json
		 FROM ost_attempts WHERE id = ? AND child_id = ? AND account_id = ?`, id, child.ChildID, child.AccountID).
		Scan(&rw.id, &rw.subject, &rw.grade, &itemsJSON, &answersJSON, &rw.started, &rw.finish, &rw.report)
	if errors.Is(err, sql.ErrNoRows) {
		return rw, ErrNotFound
	}
	if err != nil {
		return rw, err
	}
	if err := json.Unmarshal([]byte(itemsJSON), &rw.items); err != nil {
		return rw, fmt.Errorf("ost: stored items: %w", err)
	}
	rw.answers = map[string]Answer{}
	if err := json.Unmarshal([]byte(answersJSON), &rw.answers); err != nil {
		return rw, fmt.Errorf("ost: stored answers: %w", err)
	}
	return rw, nil
}

// Attempt loads one attempt for the child, including its report once graded.
func (s *Store) Attempt(ctx context.Context, child Child, id string) (Attempt, error) {
	rw, err := s.load(ctx, child, id)
	if err != nil {
		return Attempt{}, err
	}
	a := Attempt{ID: rw.id, Subject: rw.subject, Grade: rw.grade, Answers: rw.answers, StartedAt: rw.started.String}
	for _, it := range rw.items {
		a.Items = append(a.Items, it.Public())
	}
	if rw.finish.Valid {
		a.FinishedAt = rw.finish.String
		var rep Report
		if err := json.Unmarshal([]byte(rw.report), &rep); err != nil {
			return Attempt{}, fmt.Errorf("ost: stored report: %w", err)
		}
		a.Report = &rep
	}
	return a, nil
}

// SaveAnswer records one answer on an unfinished attempt. Saving is what
// makes a test resumable.
func (s *Store) SaveAnswer(ctx context.Context, child Child, id, itemID string, ans Answer, now time.Time) error {
	if len(ans.Text) > 40 || len(ans.Choices) > 8 {
		return fmt.Errorf("%w: answer too long", ErrBadRequest)
	}
	for _, c := range ans.Choices {
		if c < 0 || c > 7 {
			return fmt.Errorf("%w: bad choice", ErrBadRequest)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	// Take the write lock before reading, so the read-merge-write below
	// cannot interleave with another save or with Submit. A finished
	// attempt matches no row here.
	locked, err := s.lockOpen(ctx, tx, child, id, now)
	if err != nil {
		return err
	}
	if !locked {
		return s.openOrFinished(ctx, child, id)
	}
	var itemsJSON, answersJSON string
	err = tx.QueryRowContext(ctx,
		`SELECT items_json, answers_json FROM ost_attempts WHERE id = ? AND child_id = ? AND account_id = ?`,
		id, child.ChildID, child.AccountID).Scan(&itemsJSON, &answersJSON)
	if err != nil {
		return err
	}
	var items []Item
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return err
	}
	var it *Item
	for i := range items {
		if items[i].ID == itemID {
			it = &items[i]
		}
	}
	if it == nil {
		return fmt.Errorf("%w: no such item", ErrBadRequest)
	}
	answers := map[string]Answer{}
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return err
	}
	ans.Text = strings.TrimSpace(ans.Text)
	if it.Type == TypeNumber {
		ans.Choices = nil
	} else {
		ans.Text = ""
		for _, c := range ans.Choices {
			if c >= len(it.Choices) {
				return fmt.Errorf("%w: bad choice", ErrBadRequest)
			}
		}
		if it.Type == TypeChoice && len(ans.Choices) > 1 {
			return fmt.Errorf("%w: pick one answer", ErrBadRequest)
		}
	}
	if ans.Text == "" && len(ans.Choices) == 0 {
		delete(answers, itemID)
	} else {
		answers[itemID] = ans
	}
	out, _ := json.Marshal(answers)
	if _, err := tx.ExecContext(ctx,
		`UPDATE ost_attempts SET answers_json = ?, updated_at = ? WHERE id = ? AND child_id = ? AND account_id = ? AND finished_at IS NULL`,
		string(out), ts(now), id, child.ChildID, child.AccountID); err != nil {
		return err
	}
	return tx.Commit()
}

// lockOpen touches the child's unfinished attempt inside tx, which takes
// SQLite's write lock for the rest of the transaction. It reports false
// when the attempt is missing, belongs to someone else, or is finished.
func (s *Store) lockOpen(ctx context.Context, tx *sql.Tx, child Child, id string, now time.Time) (bool, error) {
	res, err := tx.ExecContext(ctx,
		`UPDATE ost_attempts SET updated_at = ? WHERE id = ? AND child_id = ? AND account_id = ? AND finished_at IS NULL`,
		ts(now), id, child.ChildID, child.AccountID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// openOrFinished distinguishes "not yours" from "already turned in".
func (s *Store) openOrFinished(ctx context.Context, child Child, id string) error {
	var finished sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT finished_at FROM ost_attempts WHERE id = ? AND child_id = ? AND account_id = ?`,
		id, child.ChildID, child.AccountID).Scan(&finished)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if finished.Valid {
		return ErrFinished
	}
	return ErrNotFound
}

// Submit grades an attempt. A repeated submit returns the stored report.
// The attempt is locked before its answers are read, so a save racing
// the submit either lands before grading or is refused as finished.
func (s *Store) Submit(ctx context.Context, child Child, id string, now time.Time) (Attempt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Attempt{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	locked, err := s.lockOpen(ctx, tx, child, id, now)
	if err != nil {
		return Attempt{}, err
	}
	if !locked {
		if err := s.openOrFinished(ctx, child, id); errors.Is(err, ErrFinished) {
			return s.Attempt(ctx, child, id)
		} else if err != nil {
			return Attempt{}, err
		}
		return Attempt{}, ErrNotFound
	}
	var itemsJSON, answersJSON string
	if err := tx.QueryRowContext(ctx, `SELECT items_json, answers_json FROM ost_attempts WHERE id = ?`, id).Scan(&itemsJSON, &answersJSON); err != nil {
		return Attempt{}, err
	}
	var items []Item
	answers := map[string]Answer{}
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return Attempt{}, fmt.Errorf("ost: stored items: %w", err)
	}
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return Attempt{}, fmt.Errorf("ost: stored answers: %w", err)
	}
	rep := Grade(items, answers)
	repJSON, _ := json.Marshal(rep)
	if _, err := tx.ExecContext(ctx,
		`UPDATE ost_attempts SET finished_at = ?, updated_at = ?, score = ?, total = ?, percent = ?, level = ?, report_json = ?
		 WHERE id = ? AND child_id = ? AND account_id = ? AND finished_at IS NULL`,
		ts(now), ts(now), rep.Score, rep.Total, rep.Percent, rep.Level, string(repJSON), id, child.ChildID, child.AccountID); err != nil {
		return Attempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return Attempt{}, err
	}
	return s.Attempt(ctx, child, id)
}

// Grade scores items against answers and tallies by category and standard.
func Grade(items []Item, answers map[string]Answer) Report {
	rep := Report{Total: len(items)}
	cats := map[string]*Tally{}
	stds := map[string]*Tally{}
	var catOrder, stdOrder []string
	for _, it := range items {
		given, answered := answers[it.ID]
		right := answered && Check(it, given)
		if right {
			rep.Score++
		}
		rep.Items = append(rep.Items, ItemResult{
			ID: it.ID, Category: it.Category, Standard: it.Standard, DOK: it.DOK, Type: it.Type, Prompt: it.Prompt,
			Choices: it.Choices, Answer: it.Answer, Numeric: it.Numeric, Given: given, Answered: answered, Right: right,
			Explanation: it.Explanation,
		})
		if cats[it.Category] == nil {
			cats[it.Category] = &Tally{Name: it.Category}
			catOrder = append(catOrder, it.Category)
		}
		if stds[it.Standard] == nil {
			stds[it.Standard] = &Tally{Name: it.Standard}
			stdOrder = append(stdOrder, it.Standard)
		}
		for _, t := range []*Tally{cats[it.Category], stds[it.Standard]} {
			t.Total++
			if right {
				t.Right++
			}
		}
	}
	if rep.Total > 0 {
		rep.Percent = rep.Score * 100 / rep.Total
	}
	rep.Level = LevelFor(rep.Percent)
	sort.Strings(stdOrder)
	for _, c := range catOrder {
		t := cats[c]
		t.Percent = pct(t.Right, t.Total)
		rep.Categories = append(rep.Categories, *t)
	}
	for _, st := range stdOrder {
		t := stds[st]
		t.Percent = pct(t.Right, t.Total)
		rep.Standards = append(rep.Standards, *t)
	}
	return rep
}

func pct(a, b int) int {
	if b == 0 {
		return 0
	}
	return a * 100 / b
}

// History lists a child's attempts, newest first. Unfinished attempts
// carry how many items are answered so a parent can see a test in progress.
func (s *Store) History(ctx context.Context, child Child, limit int) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, grade, started_at, finished_at, answers_json, score, total, percent, level, report_json
		 FROM ost_attempts WHERE child_id = ? AND account_id = ? AND subject = ?
		 ORDER BY started_at DESC LIMIT ?`, child.ChildID, child.AccountID, SubjectMath, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var sm Summary
		var started string
		var finished sql.NullString
		var answersJSON, reportJSON string
		if err := rows.Scan(&sm.ID, &sm.Grade, &started, &finished, &answersJSON, &sm.Score, &sm.Total, &sm.Percent, &sm.Level, &reportJSON); err != nil {
			return nil, err
		}
		sm.StartedAt, _ = time.Parse(tsLayout, started)
		var answers map[string]Answer
		if err := json.Unmarshal([]byte(answersJSON), &answers); err == nil {
			sm.Answered = len(answers)
		}
		if finished.Valid {
			sm.Finished = true
			sm.FinishedAt, _ = time.Parse(tsLayout, finished.String)
			var rep Report
			if err := json.Unmarshal([]byte(reportJSON), &rep); err == nil {
				sm.Categories, sm.Standards = rep.Categories, rep.Standards
			}
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}
