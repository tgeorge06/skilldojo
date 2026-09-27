// Package battle is the turn-based quiz battle. Each turn the server poses
// one item from the child's creature's skill; a right answer lands a hit on
// the opponent, a wrong one costs the child a heart. Three in a row charge
// a special that hits twice. No timers: speed is never scored. Opponents
// faint and rest; losing costs nothing but the battle.
package battle

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
	"github.com/tgeorge06/skilldojo/internal/kata"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var (
	ErrNotFound   = errors.New("battle: not found")
	ErrBadRequest = errors.New("battle: bad request")
	ErrNoCredit   = errors.New("battle: finish a round of five to earn a battle")
)

const (
	MaxHP         = 5
	StreakSpecial = 3
	MaxTurns      = 40 // a hard stop so a battle cannot run forever
)

const tsLayout = "2006-01-02T15:04:05.000000000Z"

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

// Store runs battles.
type Store struct {
	db     *sql.DB
	cur    *curriculum.Curriculum
	roster *kata.Roster
}

// New wraps the database, curriculum, and roster.
func New(db *sql.DB, cur *curriculum.Curriculum, roster *kata.Roster) *Store {
	return &Store{db: db, cur: cur, roster: roster}
}

// Apply is the progress.Sink that banks a credit when a round earns one.
func (s *Store) Apply(ctx context.Context, tx *sql.Tx, child progress.Child, _ []string, reward *progress.Reward, now time.Time) error {
	if !reward.BattleCredit {
		return nil
	}
	// The round id is the primary key, so a retried finish (which never
	// reaches sinks anyway) could not bank twice.
	_, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO battle_credits (round_id, account_id, child_id, earned_at) VALUES (?, ?, ?, ?)`,
		reward.RoundID, child.AccountID, child.ChildID, ts(now))
	return err
}

// Credits counts unspent battle credits.
func (s *Store) Credits(ctx context.Context, child progress.Child) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM battle_credits WHERE child_id = ? AND account_id = ? AND spent_at IS NULL`,
		child.ChildID, child.AccountID).Scan(&n)
	return n, err
}

// item is the current question. Answer is never serialised to the client.
type item struct {
	Kind     string `json:"kind"` // math | spelling
	Prompt   string `json:"prompt"`
	Clue     string `json:"clue,omitempty"`
	Sentence string `json:"sentence,omitempty"`
	Answer   string `json:"-"`
	answer   string // persisted separately in stored state
}

// Fighter is one side of the battle.
type Fighter struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Seed    int    `json:"seed"`
	Palette string `json:"palette"`
	Regions int    `json:"regions"`
	Fills   int    `json:"fills"`
	Evolved bool   `json:"evolved"`
	HP      int    `json:"hp"`
}

// Turn is one resolved exchange, kept for the battle log.
type Turn struct {
	N       int    `json:"n"`
	Prompt  string `json:"prompt"`
	Given   string `json:"given"`
	Right   bool   `json:"right"`
	Damage  int    `json:"damage"` // to the opponent (right) or to the child (wrong)
	Special bool   `json:"special"`
}

// State is the battle as stored and (minus answers) as sent.
type State struct {
	ID       string  `json:"id"`
	SkillID  string  `json:"skill_id"`
	Label    string  `json:"label"`
	Grade    int     `json:"grade"`
	Child    Fighter `json:"child"`
	Opponent Fighter `json:"opponent"`
	Turn     int     `json:"turn"`
	Streak   int     `json:"streak"`
	Item     *item   `json:"item,omitempty"`
	Log      []Turn  `json:"log"`
	Done     bool    `json:"done"`
	Won      bool    `json:"won"`
	Message  string  `json:"message"`
}

// stored adds the answer for persistence only.
type stored struct {
	State
	ItemAnswer string `json:"item_answer"`
}

// StartRequest opens a battle with one of the child's creatures.
type StartRequest struct {
	BattleID   string `json:"battle_id"`
	CreatureID string `json:"creature_id"`
}

func validID(id string) bool {
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

// Start spends one credit and opens a battle. Replaying the same battle
// id returns the battle as it stands without spending again.
func (s *Store) Start(ctx context.Context, child progress.Child, req StartRequest, now time.Time) (State, error) {
	if !validID(req.BattleID) {
		return State{}, fmt.Errorf("%w: battle_id must be 8-64 url-safe characters", ErrBadRequest)
	}
	if existing, err := s.Load(ctx, child, req.BattleID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return State{}, err
	}
	own, ok := s.roster.BySkill(req.CreatureID)
	if !ok {
		return State{}, fmt.Errorf("%w: unknown creature", ErrBadRequest)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return State{}, err
	}
	defer tx.Rollback()

	// The child must have at least found this creature.
	var state string
	var fills int
	err = tx.QueryRowContext(ctx,
		`SELECT state, fills FROM creature_state WHERE child_id = ? AND account_id = ? AND creature_id = ?`,
		child.ChildID, child.AccountID, own.ID).Scan(&state, &fills)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, fmt.Errorf("%w: find that kata first by training its skill", ErrBadRequest)
	}
	if err != nil {
		return State{}, err
	}

	// Spend exactly one unspent credit: the UPDATE is the atomic claim.
	var spent string
	err = tx.QueryRowContext(ctx,
		`UPDATE battle_credits SET spent_at = ?, battle_id = ?
		 WHERE round_id = (SELECT round_id FROM battle_credits WHERE child_id = ? AND account_id = ? AND spent_at IS NULL ORDER BY earned_at LIMIT 1)
		 RETURNING round_id`,
		ts(now), req.BattleID, child.ChildID, child.AccountID).Scan(&spent)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, ErrNoCredit
	}
	if err != nil {
		return State{}, err
	}

	opp := s.pickOpponent(own)
	st := stored{State: State{
		ID: req.BattleID, SkillID: own.SkillID, Grade: own.Grade, Label: own.Name + " vs " + opp.Name,
		Child:    Fighter{ID: own.ID, Name: own.Name, Seed: own.Seed, Palette: own.Palette, Regions: own.Regions, Fills: fills, Evolved: state == kata.StateEvolved, HP: MaxHP},
		Opponent: Fighter{ID: opp.ID, Name: opp.Name, Seed: opp.Seed, Palette: opp.Palette, Regions: opp.Regions, Fills: opp.Regions, HP: MaxHP},
		Message:  fmt.Sprintf("%s appears! Answer to strike.", opp.Name),
	}}
	if err := s.nextItem(&st, own); err != nil {
		return State{}, err
	}
	encoded, err := json.Marshal(st)
	if err != nil {
		return State{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO battles (id, account_id, child_id, state_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		req.BattleID, child.AccountID, child.ChildID, string(encoded), ts(now), ts(now)); err != nil {
		return State{}, err
	}
	if err := tx.Commit(); err != nil {
		return State{}, err
	}
	return st.State, nil
}

// pickOpponent chooses a different creature of the same grade, or any
// other creature if the grade has no other.
func (s *Store) pickOpponent(own kata.Creature) kata.Creature {
	var same, others []kata.Creature
	for _, c := range s.roster.Creatures() {
		if c.ID == own.ID {
			continue
		}
		if c.Grade == own.Grade {
			same = append(same, c)
		} else {
			others = append(others, c)
		}
	}
	pool := same
	if len(pool) == 0 {
		pool = others
	}
	return pool[rand.IntN(len(pool))]
}

// nextItem poses a fresh question from the creature's skill.
func (s *Store) nextItem(st *stored, own kata.Creature) error {
	if own.Kind == "math" {
		op := strings.TrimSuffix(strings.TrimPrefix(own.SkillID, "math-"), fmt.Sprintf("-g%d", own.Grade))
		sh, err := sheet.GenerateCount([]string{op}, own.Grade, 1)
		if err != nil {
			return err
		}
		st.Item = &item{Kind: "math", Prompt: sh.Questions[0].Prompt}
		st.ItemAnswer = sh.Answers()[0]
		return nil
	}
	var bank []curriculum.Word
	if strings.HasPrefix(own.SkillID, "sight-") {
		bank = s.cur.SightWords(own.Grade)
	} else {
		for _, w := range s.cur.Words(own.Grade) {
			if w.Skill == own.SkillID {
				bank = append(bank, w)
			}
		}
	}
	if len(bank) == 0 {
		return errors.New("battle: no words for skill " + own.SkillID)
	}
	w := bank[rand.IntN(len(bank))]
	st.Item = &item{Kind: "spelling", Prompt: "Spell the word", Clue: w.Clue, Sentence: blank(w.Sentence, w.Word)}
	st.ItemAnswer = w.Word
	return nil
}

// blank masks the first whole-word occurrence, like blankedSentence in app.js.
func blank(sentence, word string) string {
	lower := strings.ToLower(sentence)
	idx := 0
	for {
		i := strings.Index(lower[idx:], word)
		if i < 0 {
			return sentence
		}
		start := idx + i
		end := start + len(word)
		before := start == 0 || !isLetter(lower[start-1])
		after := end == len(lower) || !isLetter(lower[end])
		if before && after {
			return sentence[:start] + "_____" + sentence[end:]
		}
		idx = end
	}
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' }

// Load returns a battle as the client should see it.
func (s *Store) Load(ctx context.Context, child progress.Child, id string) (State, error) {
	st, err := s.load(ctx, child, id)
	if err != nil {
		return State{}, err
	}
	return st.State, nil
}

func (s *Store) load(ctx context.Context, child progress.Child, id string) (stored, error) {
	var raw string
	err := s.db.QueryRowContext(ctx,
		`SELECT state_json FROM battles WHERE id = ? AND child_id = ? AND account_id = ?`,
		id, child.ChildID, child.AccountID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return stored{}, ErrNotFound
	}
	if err != nil {
		return stored{}, err
	}
	var st stored
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return stored{}, err
	}
	return st, nil
}

// TurnRequest is the child's answer to the current item. Turn must match
// the battle's turn so a stale or duplicated submission is ignored.
type TurnRequest struct {
	BattleID string `json:"battle_id"`
	Turn     int    `json:"turn"`
	Answer   string `json:"answer"`
}

// Play resolves one turn.
func (s *Store) Play(ctx context.Context, child progress.Child, req TurnRequest, now time.Time) (State, error) {
	if !validID(req.BattleID) {
		return State{}, fmt.Errorf("%w: bad battle_id", ErrBadRequest)
	}
	if len(req.Answer) > 64 {
		return State{}, fmt.Errorf("%w: answer too long", ErrBadRequest)
	}
	st, err := s.load(ctx, child, req.BattleID)
	if err != nil {
		return State{}, err
	}
	if st.Done {
		return st.State, nil
	}
	if req.Turn != st.Turn {
		// Stale or duplicate: return the current state untouched.
		return st.State, nil
	}
	given := strings.TrimSpace(req.Answer)
	if given == "" {
		return State{}, fmt.Errorf("%w: type an answer first", ErrBadRequest)
	}
	right := s.check(st, given)
	turn := Turn{N: st.Turn, Prompt: st.Item.Prompt, Given: given, Right: right}
	if right {
		st.Streak++
		turn.Damage = 1
		if st.Streak >= StreakSpecial {
			turn.Damage, turn.Special = 2, true
			st.Streak = 0
		}
		st.Opponent.HP = max(0, st.Opponent.HP-turn.Damage)
		st.Message = fmt.Sprintf("Hit! %s takes %d.", st.Opponent.Name, turn.Damage)
		if turn.Special {
			st.Message = fmt.Sprintf("Special move! %s takes %d.", st.Opponent.Name, turn.Damage)
		}
	} else {
		st.Streak = 0
		turn.Damage = 1
		st.Child.HP = max(0, st.Child.HP-1)
		st.Message = fmt.Sprintf("Not quite. It was %s. %s loses a heart.", st.ItemAnswer, st.Child.Name)
	}
	st.Log = append(st.Log, turn)
	st.Turn++
	switch {
	case st.Opponent.HP == 0:
		st.Done, st.Won = true, true
		st.Message = fmt.Sprintf("%s faints and rests. %s wins!", st.Opponent.Name, st.Child.Name)
		st.Item, st.ItemAnswer = nil, ""
	case st.Child.HP == 0:
		st.Done = true
		st.Message = fmt.Sprintf("%s rests for now. Nothing lost but the battle.", st.Child.Name)
		st.Item, st.ItemAnswer = nil, ""
	case st.Turn >= MaxTurns:
		st.Done = true
		st.Message = "Both kata are tired. Call it a draw!"
		st.Item, st.ItemAnswer = nil, ""
	default:
		own, _ := s.roster.BySkill(st.SkillID)
		if err := s.nextItem(&st, own); err != nil {
			return State{}, err
		}
	}
	encoded, err := json.Marshal(st)
	if err != nil {
		return State{}, err
	}
	finished := any(nil)
	if st.Done {
		finished = ts(now)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE battles SET state_json = ?, updated_at = ?, finished_at = COALESCE(finished_at, ?)
		 WHERE id = ? AND child_id = ? AND account_id = ? AND json_extract(state_json, '$.turn') = ?`,
		string(encoded), ts(now), finished, req.BattleID, child.ChildID, child.AccountID, req.Turn)
	if err != nil {
		return State{}, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		// Lost the race with a concurrent submission for the same turn.
		return s.Load(ctx, child, req.BattleID)
	}
	return st.State, nil
}

// check compares an answer the way the game does: math through the sheet
// grader's fraction rules, spelling as the exact lowercase word.
func (s *Store) check(st stored, given string) bool {
	if st.Item == nil {
		return false
	}
	if st.Item.Kind == "math" {
		return sheet.SameAnswer(given, st.ItemAnswer)
	}
	return strings.ToLower(given) == st.ItemAnswer
}
