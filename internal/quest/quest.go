// Package quest picks three small quests a day for a child and marks them
// done as rounds finish. The picker leans on what the app already knows:
// words the child keeps missing, weak areas from practice tests, and a
// rotation through the dojo's skills. Finishing all three reveals a kata
// the child has not found yet.
package quest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tgeorge06/skilldojo/internal/kata"
	"github.com/tgeorge06/skilldojo/internal/ost"
	"github.com/tgeorge06/skilldojo/internal/progress"
)

const tsLayout = "2006-01-02T15:04:05.000000000Z"

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

// DayKey is the child's local calendar day.
func DayKey(now time.Time, tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}

// Quest is one of the day's three.
type Quest struct {
	ID     int64  `json:"id"`
	Idx    int    `json:"idx"`
	Kind   string `json:"kind"`  // math | spelling
	Focus  string `json:"focus"` // math op (addsub, mul, div, frac, tables) or spelling focus
	Table  int    `json:"table,omitempty"`
	Grade  int    `json:"grade"`
	Count  int    `json:"count"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
	Done   bool   `json:"done"`
}

// Day is what the home screen shows.
type Day struct {
	Day     string          `json:"day"`
	Quests  []Quest         `json:"quests"`
	AllDone bool            `json:"all_done"`
	Reveal  json.RawMessage `json:"reveal,omitempty"` // kata.Touched for the day's reveal
}

// Sources are the stores the picker reads. Interfaces keep tests small.
type Sources interface {
	MissedCount(ctx context.Context, child progress.Child, now time.Time) (int, error)
	History(ctx context.Context, child ost.Child, limit int) ([]ost.Summary, error)
}

// Store keeps quests.
type Store struct {
	db     *sql.DB
	roster *kata.Roster
	src    Sources
}

// New wraps the database, the creature roster (for reveals and names), and
// the progress and practice-test stores the picker reads.
func New(db *sql.DB, roster *kata.Roster, src Sources) *Store {
	return &Store{db: db, roster: roster, src: src}
}

// Today returns the child's quests for their local day, creating them on
// the first call of the day.
func (s *Store) Today(ctx context.Context, child progress.Child, now time.Time) (Day, error) {
	day := DayKey(now, child.Timezone)
	d, err := s.load(ctx, child, day)
	if err != nil {
		return Day{}, err
	}
	if len(d.Quests) == 3 {
		return d, nil
	}
	picked, err := s.pick(ctx, child, now, day)
	if err != nil {
		return Day{}, err
	}
	for i, q := range picked {
		// Two tabs starting the same day race here; the unique index keeps
		// one set and the reload below returns it.
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO quests (account_id, child_id, day, idx, kind, focus, tbl, grade, count, label, reason)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			child.AccountID, child.ChildID, day, i, q.Kind, q.Focus, q.Table, q.Grade, q.Count, q.Label, q.Reason); err != nil {
			return Day{}, err
		}
	}
	return s.load(ctx, child, day)
}

func (s *Store) load(ctx context.Context, child progress.Child, day string) (Day, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, idx, kind, focus, tbl, grade, count, label, reason, done_at, reveal_json
		 FROM quests WHERE child_id = ? AND account_id = ? AND day = ? ORDER BY idx`,
		child.ChildID, child.AccountID, day)
	if err != nil {
		return Day{}, err
	}
	defer rows.Close()
	d := Day{Day: day}
	done := 0
	for rows.Next() {
		var q Quest
		var doneAt, reveal sql.NullString
		if err := rows.Scan(&q.ID, &q.Idx, &q.Kind, &q.Focus, &q.Table, &q.Grade, &q.Count, &q.Label, &q.Reason, &doneAt, &reveal); err != nil {
			return Day{}, err
		}
		q.Done = doneAt.Valid
		if q.Done {
			done++
		}
		if reveal.Valid && reveal.String != "" {
			d.Reveal = json.RawMessage(reveal.String)
		}
		d.Quests = append(d.Quests, q)
	}
	d.AllDone = len(d.Quests) == 3 && done == 3
	return d, rows.Err()
}

var mathLabels = map[string]string{"addsub": "Add & take away", "mul": "Multiply mix", "div": "Sharing", "frac": "Fractions", "tables": "Times tables"}

// pick chooses the day's three quests. Order: a math quest, a spelling
// quest, then a third that alternates tables and sight words by day.
func (s *Store) pick(ctx context.Context, child progress.Child, now time.Time, day string) ([]Quest, error) {
	grade := child.Grade
	if grade < 1 {
		grade = 1
	}
	roundLen := child.RoundLen
	if roundLen == 0 {
		roundLen = 10
	}
	spellLen := progress.SpellingLen(roundLen)
	seed := dayNumber(day)

	// Math: the weakest practice-test area if there is one, else rotate.
	mathOp, mathReason := "", ""
	if history, err := s.src.History(ctx, ost.Child{AccountID: child.AccountID, ChildID: child.ChildID, Grade: grade}, 20); err == nil && len(history) > 0 {
		g := ost.AnalysisGrade(history, ost.NearestGrade(grade))
		an := ost.Analyze(ost.OfGrade(history, g), g)
		if len(an.Focus) > 0 {
			mathOp = opForCategory(an.Focus[0].Category, seed)
			mathReason = fmt.Sprintf("Your practice test says %s needs work", strings.ToLower(an.Focus[0].Category))
		}
	}
	if mathOp == "" {
		ops := []string{"addsub", "mul", "div", "frac"}
		if grade < 3 {
			ops = []string{"addsub", "mul", "addsub", "div"}
		}
		mathOp = ops[seed%len(ops)]
		mathReason = "Warm up your math"
	}
	q1 := Quest{Kind: "math", Focus: mathOp, Grade: grade, Count: roundLen, Label: mathLabels[mathOp], Reason: mathReason}
	if c, ok := s.roster.BySkill(fmt.Sprintf("math-%s-g%d", mathOp, grade)); ok && mathReason == "Warm up your math" {
		q1.Reason = c.Name + " is waiting"
	}

	// Spelling: review if words are waiting, else a mix.
	q2 := Quest{Kind: "spelling", Focus: "mixed", Grade: grade, Count: spellLen, Label: fmt.Sprintf("Rescue %d words", spellLen), Reason: "Listen and spell"}
	if n, err := s.src.MissedCount(ctx, child, now); err == nil && n > 0 {
		q2 = Quest{Kind: "spelling", Focus: "review", Grade: grade, Count: spellLen, Label: "Words I keep missing", Reason: fmt.Sprintf("%d words are waiting to be rescued", n)}
	}

	// Third: times tables one day, sight words the next.
	var q3 Quest
	if seed%2 == 0 && grade >= 2 {
		table := 2 + seed%11
		q3 = Quest{Kind: "math", Focus: "tables", Table: table, Grade: grade, Count: 12, Label: fmt.Sprintf("Times tables: %ds", table), Reason: "In order, nice and steady"}
		if c, ok := s.roster.BySkill(fmt.Sprintf("math-mul-g%d", grade)); ok {
			q3.Reason = c.Name + " loves the " + itoa(table) + "s"
		}
	} else {
		q3 = Quest{Kind: "spelling", Focus: "sight-words", Grade: grade, Count: spellLen, Label: "Sight words", Reason: "Words you see everywhere"}
		if c, ok := s.roster.BySkill(fmt.Sprintf("sight-g%d", grade)); ok {
			q3.Reason = c.Name + " is waiting"
		}
	}
	return []Quest{q1, q2, q3}, nil
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// dayNumber turns YYYY-MM-DD into a small integer that changes daily.
func dayNumber(day string) int {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return 0
	}
	return int(t.Unix() / 86400)
}

func opForCategory(category string, seed int) string {
	switch category {
	case "Multiplication and Division":
		if seed%2 == 0 {
			return "tables"
		}
		return "mul"
	case "Fractions", "Decimals":
		return "frac"
	case "Number and Operations":
		return "addsub"
	}
	return "mul"
}

// Apply is the progress.Sink: a finished round completes the first open
// quest it matches; the third completion reveals a kata. It runs after the
// kata sink so the reveal joins the round's creatures.
func (s *Store) Apply(ctx context.Context, tx *sql.Tx, child progress.Child, _ []string, reward *progress.Reward, now time.Time) error {
	var kind, focus string
	if err := tx.QueryRowContext(ctx, `SELECT kind, focus FROM rounds WHERE id = ? AND child_id = ? AND account_id = ?`,
		reward.RoundID, child.ChildID, child.AccountID).Scan(&kind, &focus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	day := DayKey(now, child.Timezone)
	rows, err := tx.QueryContext(ctx,
		`SELECT id, idx, kind, focus, tbl, done_at IS NOT NULL FROM quests WHERE child_id = ? AND account_id = ? AND day = ? ORDER BY idx`,
		child.ChildID, child.AccountID, day)
	if err != nil {
		return err
	}
	var open []Quest
	total, done := 0, 0
	for rows.Next() {
		var q Quest
		if err := rows.Scan(&q.ID, &q.Idx, &q.Kind, &q.Focus, &q.Table, &q.Done); err != nil {
			rows.Close()
			return err
		}
		total++
		if q.Done {
			done++
		} else {
			open = append(open, q)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var hit *Quest
	for i := range open {
		if Matches(open[i], kind, focus) {
			hit = &open[i]
			break
		}
	}
	if hit == nil {
		return nil
	}
	var revealJSON any
	if total == 3 && done+1 == 3 {
		t, err := s.reveal(ctx, tx, child, now)
		if err != nil {
			return err
		}
		if t != nil {
			encoded, _ := json.Marshal(t)
			revealJSON = string(encoded)
			var touched []kata.Touched
			if len(reward.Creatures) > 0 {
				_ = json.Unmarshal(reward.Creatures, &touched)
			}
			touched = append(touched, *t)
			reward.Creatures, _ = json.Marshal(touched)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE quests SET done_at = ?, round_id = ?, reveal_json = ? WHERE id = ? AND child_id = ? AND account_id = ? AND done_at IS NULL`,
		ts(now), reward.RoundID, revealJSON, hit.ID, child.ChildID, child.AccountID)
	return err
}

// Matches reports whether a finished round of kind/focus completes q.
// A math quest matches a round that practiced its op (a tables quest,
// any table); a spelling quest matches its focus, and a "mixed" quest
// matches any spelling round.
func Matches(q Quest, kind, focus string) bool {
	if q.Kind != kind {
		return false
	}
	if kind == "math" {
		if q.Focus == "tables" {
			return strings.HasPrefix(focus, "tables:")
		}
		for _, op := range strings.Split(focus, ",") {
			if op == q.Focus {
				return true
			}
		}
		return false
	}
	return q.Focus == "mixed" || q.Focus == focus
}

// reveal gives two colors to a creature of the child's grade they have
// not found yet, so the day's reward is a new face. Nil when every
// creature at that grade is already found.
func (s *Store) reveal(ctx context.Context, tx *sql.Tx, child progress.Child, now time.Time) (*kata.Touched, error) {
	seen := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT creature_id FROM creature_state WHERE child_id = ? AND account_id = ?`, child.ChildID, child.AccountID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		seen[id] = true
	}
	rows.Close()
	var pick *kata.Creature
	for _, c := range s.roster.Creatures() {
		if c.Grade == child.Grade && !seen[c.ID] {
			cc := c
			pick = &cc
			break
		}
	}
	if pick == nil {
		return nil, nil
	}
	fills := min(2, pick.Regions)
	// creature_state timestamps follow the kata store's layout.
	kts := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO creature_state (account_id, child_id, creature_id, state, fills, seen_at, updated_at) VALUES (?, ?, ?, 'seen', ?, ?, ?)`,
		child.AccountID, child.ChildID, pick.ID, fills, kts, kts); err != nil {
		return nil, err
	}
	return &kata.Touched{ID: pick.ID, Name: pick.Name, Seed: pick.Seed, Palette: pick.Palette, Regions: pick.Regions, Fills: fills, FillsAdded: fills, State: kata.StateSeen, NewlySeen: true}, nil
}
