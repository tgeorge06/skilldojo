// Package progress records practice rounds for signed-in children, grades
// them on the server, tracks per-skill mastery with Leitner boxes, and
// computes the reward a finished round earns. It is the only place rewards
// are decided; the client posts what the child did, never how it went.
package progress

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var (
	// ErrNotFound covers rounds that do not exist or belong to someone else.
	ErrNotFound = errors.New("progress: round not found")
	// ErrBadRequest is a validation failure safe to show the client.
	ErrBadRequest = errors.New("progress: bad request")
)

const (
	KindMath     = "math"
	KindSpelling = "spelling"

	// FocusMixed rotates through a grade's spelling skills; FocusReview
	// drills the words this child keeps missing.
	FocusMixed  = "mixed"
	FocusReview = "review"

	// MaxBox is the top Leitner box.
	MaxBox = 4
	// MissedWindow is how far back "words you keep missing" looks.
	MissedWindow = 30 * 24 * time.Hour
	// EvolveDays is the minimum span between the first successful review
	// and evolution; playing more today cannot shorten it.
	EvolveDays = 14
	// MasteryThreshold is the per-skill accuracy that counts as a successful review.
	MasteryThreshold = 0.8
)

// leitnerIntervals are the review gaps per box, in days.
var leitnerIntervals = [MaxBox + 1]int{1, 3, 7, 14, 30}

// Store is the data access layer.
type Store struct {
	db     *sql.DB
	cur    *curriculum.Curriculum
	sheets *sheet.Store
	sink   Sink
}

// New wraps the database, curriculum, and the in-memory sheet store the
// math dojo already uses.
func New(db *sql.DB, cur *curriculum.Curriculum, sheets *sheet.Store) *Store {
	return &Store{db: db, cur: cur, sheets: sheets}
}

// StartRequest is what the client sends to open a round.
type StartRequest struct {
	RoundID string   `json:"round_id"`
	Kind    string   `json:"kind"`
	Focus   string   `json:"focus"` // spelling: skill id, "mixed", "sight-words", "review"
	Ops     []string `json:"ops"`   // math
	Grade   int      `json:"grade"`
	Count   int      `json:"count"`
}

// StartResponse carries the material for the round. Math answers stay on
// the server; spelling words are sent because the game reveals them anyway.
type StartResponse struct {
	RoundID   string            `json:"round_id"`
	Kind      string            `json:"kind"`
	Words     []curriculum.Word `json:"words,omitempty"`
	SheetID   string            `json:"sheet_id,omitempty"`
	Questions []sheet.Question  `json:"questions,omitempty"`
}

// FinishRequest is the client's record of what happened.
type FinishRequest struct {
	RoundID string               `json:"round_id"`
	Guesses [][]curriculum.Guess `json:"guesses,omitempty"` // spelling, one list per word
	Answers []string             `json:"answers,omitempty"` // math
}

// WordResult is one graded spelling word.
type WordResult struct {
	Word     string `json:"word"`
	Won      bool   `json:"won"`
	Mistakes int    `json:"mistakes"`
}

// Reward is everything a finished round earned. This is the arithmetic;
// a Sink (the creature collection) records what it did in Creatures.
type Reward struct {
	Fills        int             `json:"fills"`          // creature energy
	FillsBySkill map[string]int  `json:"fills_by_skill"` // per skill id, for the creature bound to each
	MosaicCells  int             `json:"mosaic_cells"`
	BattleCredit bool            `json:"battle_credit"`
	ReviewDue    int             `json:"review_due"` // words the child keeps missing
	Evolved      []string        `json:"evolved"`    // skill ids that reached mastery in this round
	Creatures    json.RawMessage `json:"creatures,omitempty"`
}

// Sink is applied inside the finishing transaction, once per round, with
// the ordered list of skills the round touched. It may annotate the reward.
type Sink interface {
	Apply(ctx context.Context, tx *sql.Tx, child Child, skills []string, reward *Reward, now time.Time) error
}

// SetSink registers the reward consumer.
func (s *Store) SetSink(sink Sink) { s.sink = sink }

// FinishResponse is returned to the client and stored verbatim so a retry
// returns the identical payload.
type FinishResponse struct {
	RoundID     string         `json:"round_id"`
	Kind        string         `json:"kind"`
	Score       int            `json:"score"`
	Total       int            `json:"total"`
	Percent     int            `json:"percent"`
	Results     []sheet.Result `json:"results,omitempty"`
	WordResults []WordResult   `json:"word_results,omitempty"`
	Reward      Reward         `json:"reward"`
}

// Child is the profile a round belongs to.
type Child struct {
	AccountID int64
	ChildID   int64
	Grade     int
	Timezone  string
}

// tsLayout is fixed width so lexical comparison in SQL is chronological;
// RFC3339Nano drops trailing zeros and "05Z" would sort after "05.5Z".
const tsLayout = "2006-01-02T15:04:05.000000000Z"

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

// localDate is the child's calendar date, which keys review scheduling.
func localDate(now time.Time, tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc = time.UTC
	}
	return now.In(loc).Format("2006-01-02")
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func validRoundID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// MathSkill is the skill id for one math item.
func MathSkill(op string, grade int) string { return fmt.Sprintf("math-%s-g%d", op, grade) }

// SightSkill is the skill id for a sight word of a given band.
func SightSkill(grade int) string { return fmt.Sprintf("sight-g%d", grade) }

// Start opens a round for the child. It is idempotent on RoundID for the
// same child: replaying a start returns the same words or sheet.
func (s *Store) Start(ctx context.Context, child Child, req StartRequest, now time.Time) (StartResponse, error) {
	if !validRoundID(req.RoundID) {
		return StartResponse{}, fmt.Errorf("%w: round_id must be 8-64 url-safe characters", ErrBadRequest)
	}
	if req.Grade < 1 || req.Grade > 5 {
		return StartResponse{}, fmt.Errorf("%w: grade must be 1-5", ErrBadRequest)
	}
	if existing, err := s.existingStart(ctx, child, req.RoundID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return StartResponse{}, err
	}
	var resp StartResponse
	var err error
	switch req.Kind {
	case KindSpelling:
		resp, err = s.startSpelling(ctx, child, req, now)
	case KindMath:
		resp, err = s.startMath(ctx, child, req, now)
	default:
		return StartResponse{}, fmt.Errorf("%w: kind must be math or spelling", ErrBadRequest)
	}
	if err != nil && isUniqueViolation(err) {
		// Two concurrent starts for the same id: the loser replays the winner.
		return s.existingStart(ctx, child, req.RoundID)
	}
	return resp, err
}

// isUniqueViolation recognises SQLite's primary-key conflict. The round id
// is the primary key, and a conflict from another child's round of the same
// id surfaces as ErrNotFound from existingStart, never as their data.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *Store) existingStart(ctx context.Context, child Child, roundID string) (StartResponse, error) {
	var kind string
	var sheetID sql.NullString
	var finished sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT kind, sheet_id, finished_at FROM rounds WHERE id = ? AND account_id = ? AND child_id = ?`,
		roundID, child.AccountID, child.ChildID).Scan(&kind, &sheetID, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return StartResponse{}, ErrNotFound
	}
	if err != nil {
		return StartResponse{}, err
	}
	if finished.Valid {
		return StartResponse{}, fmt.Errorf("%w: round already finished", ErrBadRequest)
	}
	resp := StartResponse{RoundID: roundID, Kind: kind}
	if kind == KindSpelling {
		rows, err := s.db.QueryContext(ctx,
			`SELECT item_key, skill_id FROM round_items WHERE round_id = ? AND account_id = ? AND child_id = ? ORDER BY idx`,
			roundID, child.AccountID, child.ChildID)
		if err != nil {
			return StartResponse{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var key, skill string
			if err := rows.Scan(&key, &skill); err != nil {
				return StartResponse{}, err
			}
			if w, ok := s.wordFor(key, skill); ok {
				resp.Words = append(resp.Words, w)
			}
		}
		return resp, rows.Err()
	}
	// A math sheet lives in memory; if the server restarted it is gone and
	// the client must start a new round.
	sh, ok := s.sheets.Peek(sheetID.String)
	if !ok {
		return StartResponse{}, fmt.Errorf("%w: sheet expired, start a new round", ErrBadRequest)
	}
	resp.SheetID, resp.Questions = sh.ID, sh.Questions
	return resp, nil
}

func (s *Store) startSpelling(ctx context.Context, child Child, req StartRequest, now time.Time) (StartResponse, error) {
	if req.Count != 5 && req.Count != 10 {
		return StartResponse{}, fmt.Errorf("%w: count must be 5 or 10", ErrBadRequest)
	}
	focus := req.Focus
	if focus == "" {
		focus = FocusMixed
	}
	var missed []MissedWord
	if focus == FocusReview {
		var err error
		if missed, err = s.MissedWords(ctx, child, now); err != nil {
			return StartResponse{}, err
		}
		if len(missed) == 0 {
			return StartResponse{}, fmt.Errorf("%w: nothing to review yet", ErrBadRequest)
		}
	} else if focus != FocusMixed && focus != curriculum.SightWordSkill {
		if _, ok := s.cur.Skill(focus); !ok {
			return StartResponse{}, fmt.Errorf("%w: unknown focus", ErrBadRequest)
		}
	}
	words := s.selectWords(focus, req.Grade, req.Count, missed)
	if len(words) == 0 {
		return StartResponse{}, fmt.Errorf("%w: no words for that focus", ErrBadRequest)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StartResponse{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rounds (id, account_id, child_id, kind, focus, grade, child_grade, started_at)
		 VALUES (?, ?, ?, 'spelling', ?, ?, ?, ?)`,
		req.RoundID, child.AccountID, child.ChildID, focus, req.Grade, child.Grade, ts(now)); err != nil {
		return StartResponse{}, err
	}
	for i, w := range words {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO round_items (round_id, idx, account_id, child_id, item_key, skill_id, grade)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			req.RoundID, i, child.AccountID, child.ChildID, w.Word, s.wordSkill(w), s.wordGrade(w, req.Grade)); err != nil {
			return StartResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return StartResponse{}, err
	}
	return StartResponse{RoundID: req.RoundID, Kind: KindSpelling, Words: words}, nil
}

// wordFor resolves a recorded item back to its curriculum entry using the
// skill it was recorded under, so a word present in both banks comes back
// from the right one.
func (s *Store) wordFor(word, skill string) (curriculum.Word, bool) {
	if strings.HasPrefix(skill, "sight-") {
		return s.cur.SightWord(word)
	}
	return s.cur.Word(word)
}

// wordSkill maps a word to the skill (and creature) it trains. Sight words
// are banded by grade so each band has its own creature.
func (s *Store) wordSkill(w curriculum.Word) string {
	if w.Skill == curriculum.SightWordSkill {
		return SightSkill(s.sightBand(w.Word))
	}
	return w.Skill
}

func (s *Store) wordGrade(w curriculum.Word, roundGrade int) int {
	if w.Skill == curriculum.SightWordSkill {
		return s.sightBand(w.Word)
	}
	// Pattern skill ids start with g<N>-.
	if len(w.Skill) > 2 && w.Skill[0] == 'g' && w.Skill[1] >= '1' && w.Skill[1] <= '5' {
		return int(w.Skill[1] - '0')
	}
	return roundGrade
}

func (s *Store) sightBand(word string) int {
	for _, g := range s.cur.Grades() {
		for _, w := range s.cur.SightWords(g) {
			if w.Word == word {
				return g
			}
		}
	}
	return 1
}

func (s *Store) shuffle(words []curriculum.Word) []curriculum.Word {
	out := append([]curriculum.Word(nil), words...)
	// Package-level rand is goroutine-safe; a shared *rand.Rand is not.
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// selectWords ports selectSpellingWords from app.js: a 60/40 band mix for
// sight words, focused-then-review for one skill, round-robin across skill
// buckets for mixed, and the missed list for review.
func (s *Store) selectWords(focus string, grade, count int, missed []MissedWord) []curriculum.Word {
	if focus == FocusReview {
		var words []curriculum.Word
		for _, m := range missed {
			if w, ok := s.wordFor(m.Word, m.Skill); ok {
				words = append(words, w)
			}
		}
		words = s.shuffle(words)
		if len(words) > count {
			words = words[:count]
		}
		return words
	}
	bank := s.cur.Words(grade)
	if focus == curriculum.SightWordSkill {
		current := s.shuffle(s.cur.SightWords(grade))
		var earlier []curriculum.Word
		for _, g := range s.cur.Grades() {
			if g < grade {
				earlier = append(earlier, s.cur.SightWords(g)...)
			}
		}
		earlier = s.shuffle(earlier)
		if len(earlier) == 0 {
			return current[:min(count, len(current))]
		}
		currentCount := min(len(current), (count*6+9)/10) // ceil(count*0.6)
		picked := append(current[:currentCount], earlier[:min(len(earlier), count-currentCount)]...)
		return s.shuffle(picked)
	}
	if focus != FocusMixed {
		var focused, review []curriculum.Word
		for _, w := range bank {
			if w.Skill == focus {
				focused = append(focused, w)
			} else {
				review = append(review, w)
			}
		}
		all := append(s.shuffle(focused), s.shuffle(review)...)
		return all[:min(count, len(all))]
	}
	skills := s.cur.Skills(grade)
	buckets := make([][]curriculum.Word, 0, len(skills))
	for _, sk := range skills {
		var b []curriculum.Word
		for _, w := range bank {
			if w.Skill == sk.ID {
				b = append(b, w)
			}
		}
		buckets = append(buckets, s.shuffle(b))
	}
	rand.Shuffle(len(buckets), func(i, j int) { buckets[i], buckets[j] = buckets[j], buckets[i] })
	var selected []curriculum.Word
	for round := 0; len(selected) < count; round++ {
		added := false
		for _, b := range buckets {
			if round < len(b) && len(selected) < count {
				selected = append(selected, b[round])
				added = true
			}
		}
		if !added {
			break
		}
	}
	return selected
}

func (s *Store) startMath(ctx context.Context, child Child, req StartRequest, now time.Time) (StartResponse, error) {
	sh, err := sheet.Generate(req.Ops, req.Grade, req.Count)
	if err != nil {
		return StartResponse{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StartResponse{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rounds (id, account_id, child_id, kind, focus, grade, child_grade, sheet_id, started_at)
		 VALUES (?, ?, ?, 'math', ?, ?, ?, ?, ?)`,
		req.RoundID, child.AccountID, child.ChildID, strings.Join(req.Ops, ","), req.Grade, child.Grade, sh.ID, ts(now)); err != nil {
		return StartResponse{}, err
	}
	for i, q := range sh.Questions {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO round_items (round_id, idx, account_id, child_id, item_key, skill_id, grade)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			req.RoundID, i, child.AccountID, child.ChildID, q.Prompt, MathSkill(q.Op, req.Grade), req.Grade); err != nil {
			return StartResponse{}, err
		}
	}
	// The sheet must exist before the round is visible, or a concurrent
	// replay of the same id could see the row and find no sheet. Store it
	// first; if the commit fails, take it back out.
	if err := s.sheets.Put(sh); err != nil {
		return StartResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		s.sheets.Remove(sh.ID)
		return StartResponse{}, err
	}
	return StartResponse{RoundID: req.RoundID, Kind: KindMath, SheetID: sh.ID, Questions: sh.Questions}, nil
}

type itemRow struct {
	idx       int
	key       string
	skill     string
	grade     int
	correct   bool
	attempted bool // at least one guess or a non-blank answer
}

// Finish grades the round on the server, records every item, updates
// mastery, and returns the reward. A second call for the same round
// returns the stored response unchanged.
func (s *Store) Finish(ctx context.Context, child Child, req FinishRequest, now time.Time) (FinishResponse, error) {
	if !validRoundID(req.RoundID) {
		return FinishResponse{}, fmt.Errorf("%w: bad round_id", ErrBadRequest)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FinishResponse{}, err
	}
	defer tx.Rollback()

	var kind string
	var grade, childGrade int
	var sheetID, finished, stored sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT kind, grade, child_grade, sheet_id, finished_at, result_json FROM rounds
		 WHERE id = ? AND account_id = ? AND child_id = ?`,
		req.RoundID, child.AccountID, child.ChildID).Scan(&kind, &grade, &childGrade, &sheetID, &finished, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		return FinishResponse{}, ErrNotFound
	}
	if err != nil {
		return FinishResponse{}, err
	}
	if finished.Valid {
		var resp FinishResponse
		if err := json.Unmarshal([]byte(stored.String), &resp); err != nil {
			return FinishResponse{}, err
		}
		return resp, nil
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT idx, item_key, skill_id, grade FROM round_items WHERE round_id = ? AND account_id = ? AND child_id = ? ORDER BY idx`,
		req.RoundID, child.AccountID, child.ChildID)
	if err != nil {
		return FinishResponse{}, err
	}
	var items []itemRow
	for rows.Next() {
		var it itemRow
		if err := rows.Scan(&it.idx, &it.key, &it.skill, &it.grade); err != nil {
			rows.Close()
			return FinishResponse{}, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return FinishResponse{}, err
	}

	resp := FinishResponse{RoundID: req.RoundID, Kind: kind, Total: len(items)}
	switch kind {
	case KindSpelling:
		if len(req.Guesses) != len(items) {
			return FinishResponse{}, fmt.Errorf("%w: expected guesses for %d words", ErrBadRequest, len(items))
		}
		for i := range items {
			if len(req.Guesses[i]) > 64 {
				return FinishResponse{}, fmt.Errorf("%w: too many guesses", ErrBadRequest)
			}
			out := curriculum.Replay(items[i].key, req.Guesses[i])
			items[i].correct = out.Won
			items[i].attempted = len(req.Guesses[i]) > 0
			resp.WordResults = append(resp.WordResults, WordResult{Word: items[i].key, Won: out.Won, Mistakes: out.Mistakes})
		}
	case KindMath:
		// Evaluate does not consume the sheet; it is removed only after the
		// round commits, so a failed write leaves the round finishable.
		results, err := s.sheets.Evaluate(sheetID.String, req.Answers)
		if err != nil {
			return FinishResponse{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
		}
		if len(results) != len(items) {
			return FinishResponse{}, errors.New("progress: sheet and round disagree")
		}
		for i := range items {
			items[i].correct = results[i].Right
			items[i].attempted = strings.TrimSpace(req.Answers[i]) != ""
		}
		resp.Results = results
	}
	for _, it := range items {
		if it.correct {
			resp.Score++
		}
	}
	if resp.Total > 0 {
		resp.Percent = resp.Score * 100 / resp.Total
	}

	// Which of these items had the child missed before? Read before writing
	// this round's items so today's misses do not count as "previous".
	previouslyMissed, err := missedBefore(ctx, tx, child, items, now)
	if err != nil {
		return FinishResponse{}, err
	}
	for _, it := range items {
		if _, err := tx.ExecContext(ctx,
			`UPDATE round_items SET correct = ?, answered_at = ? WHERE round_id = ? AND idx = ? AND account_id = ? AND child_id = ?`,
			boolInt(it.correct), ts(now), req.RoundID, it.idx, child.AccountID, child.ChildID); err != nil {
			return FinishResponse{}, err
		}
	}

	today := localDate(now, child.Timezone)
	evolved, err := s.updateMastery(ctx, tx, child, items, today, now)
	if err != nil {
		return FinishResponse{}, err
	}
	resp.Reward = computeReward(items, childGrade, previouslyMissed)
	resp.Reward.Evolved = evolved
	if resp.Reward.ReviewDue, err = countMissed(ctx, tx, child, now); err != nil {
		return FinishResponse{}, err
	}
	if s.sink != nil {
		// Only skills the child actually attempted count as practiced; a
		// blank submission must not mark creatures seen.
		var skills []string
		for _, it := range items {
			if it.attempted {
				skills = append(skills, it.skill)
			}
		}
		if err := s.sink.Apply(ctx, tx, child, skills, &resp.Reward, now); err != nil {
			return FinishResponse{}, err
		}
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		return FinishResponse{}, err
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE rounds SET finished_at = ?, correct = ?, total = ?, result_json = ?
		 WHERE id = ? AND account_id = ? AND child_id = ? AND finished_at IS NULL`,
		ts(now), resp.Score, resp.Total, string(encoded), req.RoundID, child.AccountID, child.ChildID)
	if err != nil {
		return FinishResponse{}, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return FinishResponse{}, errors.New("progress: round finished concurrently")
	}
	if err := tx.Commit(); err != nil {
		return FinishResponse{}, err
	}
	if kind == KindMath {
		s.sheets.Remove(sheetID.String)
	}
	return resp, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// computeReward is the single reward function. Fills weight each correct
// item by where it sits relative to the child's grade and whether the child
// had missed it before; anything below the child's grade earns nothing so
// easy mode cannot be farmed. Mosaic cells count every correct answer.
func computeReward(items []itemRow, childGrade int, previouslyMissed map[string]bool) Reward {
	r := Reward{FillsBySkill: map[string]int{}}
	attempted := 0
	for _, it := range items {
		if it.attempted {
			attempted++
		}
		if !it.correct {
			continue
		}
		r.MosaicCells++
		weight := 0
		switch {
		case it.grade < childGrade:
			weight = 0 // never, even for a previously missed word: no easy-mode farming
		case it.grade > childGrade || previouslyMissed[it.key]:
			weight = 2
		default:
			weight = 1
		}
		r.Fills += weight
		if weight > 0 {
			r.FillsBySkill[it.skill] += weight
		}
	}
	// A credit is for playing, not for being assigned five items.
	r.BattleCredit = attempted >= 5
	return r
}

// missedBefore marks items whose most recent answered attempt inside the
// missed window was wrong. A word answered correctly since then, or missed
// long ago, is ordinary practice again, so a single old miss cannot be
// farmed for double credit forever. One query for the whole round.
func missedBefore(ctx context.Context, tx *sql.Tx, child Child, items []itemRow, now time.Time) (map[string]bool, error) {
	out := map[string]bool{}
	if len(items) == 0 {
		return out, nil
	}
	args := []any{child.ChildID, child.AccountID, ts(now.Add(-MissedWindow))}
	placeholders := make([]string, len(items))
	for i, it := range items {
		placeholders[i] = "?"
		args = append(args, it.key)
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT item_key FROM (
		   SELECT item_key, correct,
		          ROW_NUMBER() OVER (PARTITION BY item_key ORDER BY answered_at DESC) AS rn
		   FROM round_items
		   WHERE child_id = ? AND account_id = ? AND answered_at > ? AND correct IS NOT NULL
		     AND item_key IN (`+strings.Join(placeholders, ",")+`)
		 ) WHERE rn = 1 AND correct = 0`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out[key] = true
	}
	return out, rows.Err()
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// MissedWord is a spelling item the child keeps missing, with the skill it
// was recorded under so it resolves to the right curriculum bank.
type MissedWord struct {
	Word  string `json:"word"`
	Skill string `json:"skill"`
}

// missedWords lists spelling words with two or more misses inside the
// window whose most recent attempt was also a miss, computed in SQL.
func missedWords(ctx context.Context, q queryer, child Child, now time.Time) ([]MissedWord, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT item_key, skill_id FROM (
		   SELECT item_key, skill_id, correct,
		          ROW_NUMBER() OVER (PARTITION BY item_key ORDER BY answered_at DESC) AS rn,
		          SUM(CASE WHEN correct = 0 THEN 1 ELSE 0 END) OVER (PARTITION BY item_key) AS misses
		   FROM round_items
		   WHERE child_id = ? AND account_id = ? AND answered_at > ? AND correct IS NOT NULL
		     AND skill_id NOT LIKE 'math-%'
		 ) WHERE rn = 1 AND correct = 0 AND misses >= 2
		 ORDER BY item_key`,
		child.ChildID, child.AccountID, ts(now.Add(-MissedWindow)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissedWord
	for rows.Next() {
		var m MissedWord
		if err := rows.Scan(&m.Word, &m.Skill); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func countMissed(ctx context.Context, tx *sql.Tx, child Child, now time.Time) (int, error) {
	words, err := missedWords(ctx, tx, child, now)
	return len(words), err
}

// MissedCount is the bounded count path for the index: the same window
// query, aggregated in SQL.
func (s *Store) MissedCount(ctx context.Context, child Child, now time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (
		   SELECT item_key, correct,
		          ROW_NUMBER() OVER (PARTITION BY item_key ORDER BY answered_at DESC) AS rn,
		          SUM(CASE WHEN correct = 0 THEN 1 ELSE 0 END) OVER (PARTITION BY item_key) AS misses
		   FROM round_items
		   WHERE child_id = ? AND account_id = ? AND answered_at > ? AND correct IS NOT NULL
		     AND skill_id NOT LIKE 'math-%'
		 ) WHERE rn = 1 AND correct = 0 AND misses >= 2`,
		child.ChildID, child.AccountID, ts(now.Add(-MissedWindow))).Scan(&n)
	return n, err
}

// MissedWords is the public form, used for the review focus and the index.
func (s *Store) MissedWords(ctx context.Context, child Child, now time.Time) ([]MissedWord, error) {
	return missedWords(ctx, s.db, child, now)
}

// SkillProgress is one child's standing in one skill.
type SkillProgress struct {
	SkillID   string
	Box       int
	DueOn     string
	ReviewsOK int
	FirstOKOn string
	EvolvedOn string // local date the skill was first observed evolved
	Rounds    int
	Evolved   bool
}

// Evolved is the mastery rule: box 3 or higher, two successful reviews,
// and at least EvolveDays between the first success and today.
func (p SkillProgress) evolvedOn(today string) bool {
	if p.Box < 3 || p.ReviewsOK < 2 || p.FirstOKOn == "" {
		return false
	}
	return parseDate(today).Sub(parseDate(p.FirstOKOn)) >= EvolveDays*24*time.Hour
}

// updateMastery applies the Leitner rule per skill present in the round and
// returns the skills that crossed into "evolved" during this update.
func (s *Store) updateMastery(ctx context.Context, tx *sql.Tx, child Child, items []itemRow, today string, now time.Time) ([]string, error) {
	type tally struct{ correct, total int }
	bySkill := map[string]*tally{}
	var order []string
	for _, it := range items {
		t, ok := bySkill[it.skill]
		if !ok {
			t = &tally{}
			bySkill[it.skill] = t
			order = append(order, it.skill)
		}
		t.total++
		if it.correct {
			t.correct++
		}
	}
	var evolved []string
	for _, skill := range order {
		t := bySkill[skill]
		p := SkillProgress{SkillID: skill, DueOn: today}
		var firstOK, evolvedOn sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT box, due_on, reviews_ok, first_ok_on, rounds, evolved_on FROM skill_progress WHERE child_id = ? AND account_id = ? AND skill_id = ?`,
			child.ChildID, child.AccountID, skill).Scan(&p.Box, &p.DueOn, &p.ReviewsOK, &firstOK, &p.Rounds, &evolvedOn)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		p.FirstOKOn, p.EvolvedOn = firstOK.String, evolvedOn.String
		// The calendar may have crossed the line since the last round; note
		// it before this round's result can drop the box.
		evolvedByCalendar := p.evolvedOn(today)

		passed := float64(t.correct)/float64(t.total) >= MasteryThreshold
		due := today >= p.DueOn
		switch {
		case passed && due:
			p.Box = min(MaxBox, p.Box+1)
			p.ReviewsOK++
			if p.FirstOKOn == "" {
				p.FirstOKOn = today
			}
			p.DueOn = parseDate(today).AddDate(0, 0, leitnerIntervals[p.Box]).Format("2006-01-02")
		case passed:
			// Early success keeps the schedule; it cannot be rushed.
		default:
			p.Box = max(0, p.Box-1)
			p.DueOn = parseDate(today).AddDate(0, 0, leitnerIntervals[p.Box]).Format("2006-01-02")
		}
		p.Rounds++
		// Evolution is observed here, whether the round or the calendar
		// crossed the line, and recorded so it is reported exactly once.
		newlyEvolved := p.EvolvedOn == "" && (evolvedByCalendar || p.evolvedOn(today))
		if newlyEvolved {
			p.EvolvedOn = today
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO skill_progress (account_id, child_id, skill_id, box, due_on, reviews_ok, first_ok_on, last_seen_at, rounds, evolved_on)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(child_id, skill_id) DO UPDATE SET
			   box = excluded.box, due_on = excluded.due_on, reviews_ok = excluded.reviews_ok,
			   first_ok_on = excluded.first_ok_on, last_seen_at = excluded.last_seen_at, rounds = excluded.rounds,
			   evolved_on = excluded.evolved_on`,
			child.AccountID, child.ChildID, skill, p.Box, p.DueOn, p.ReviewsOK, nullIfEmpty(p.FirstOKOn), ts(now), p.Rounds, nullIfEmpty(p.EvolvedOn)); err != nil {
			return nil, err
		}
		if newlyEvolved {
			evolved = append(evolved, skill)
		}
	}
	return evolved, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Progress lists every skill the child has touched, with the evolved flag
// evaluated for today.
func (s *Store) Progress(ctx context.Context, child Child, now time.Time) ([]SkillProgress, error) {
	today := localDate(now, child.Timezone)
	rows, err := s.db.QueryContext(ctx,
		`SELECT skill_id, box, due_on, reviews_ok, first_ok_on, rounds, evolved_on FROM skill_progress
		 WHERE child_id = ? AND account_id = ? ORDER BY skill_id`, child.ChildID, child.AccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillProgress
	for rows.Next() {
		var p SkillProgress
		var firstOK, evolvedOn sql.NullString
		if err := rows.Scan(&p.SkillID, &p.Box, &p.DueOn, &p.ReviewsOK, &firstOK, &p.Rounds, &evolvedOn); err != nil {
			return nil, err
		}
		p.FirstOKOn, p.EvolvedOn = firstOK.String, evolvedOn.String
		// Once evolved, always evolved: a later bad round drops the box but
		// does not un-evolve the creature.
		p.Evolved = p.EvolvedOn != "" || p.evolvedOn(today)
		out = append(out, p)
	}
	return out, rows.Err()
}
