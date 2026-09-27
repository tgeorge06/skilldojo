// Package kata is the creature collection. Every creature is bound to one
// curriculum skill at one grade; finding it means practicing that skill,
// catching it means coloring all its regions, and evolving it means the
// skill reached mastery in progress. The roster is static data; only each
// child's discovery state is stored.
package kata

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/progress"
)

//go:embed kata.json
var rosterJSON []byte

// Creature is one roster entry.
type Creature struct {
	ID           string `json:"id"`
	SkillID      string `json:"skill_id"`
	Kind         string `json:"kind"` // math | spelling
	Grade        int    `json:"grade"`
	Name         string `json:"name"`
	HintVague    string `json:"hint_vague"`
	HintSpecific string `json:"hint_specific"`
	Seed         int    `json:"seed"`
	Regions      int    `json:"regions"`
	Palette      string `json:"palette"`
}

// Roster is the parsed, validated creature list.
type Roster struct {
	creatures []Creature
	bySkill   map[string]Creature
}

type rosterFile struct {
	Creatures []Creature `json:"creatures"`
}

// States, in order of progress.
const (
	StateUnknown = "unknown"
	StateSeen    = "seen"
	StateCaught  = "caught"
	StateEvolved = "evolved"
)

var idShape = regexp.MustCompile(`^[a-z0-9-]+$`)

// Load parses and validates the embedded roster against the curriculum.
func Load(cur *curriculum.Curriculum) (*Roster, error) {
	var f rosterFile
	if err := json.Unmarshal(rosterJSON, &f); err != nil {
		return nil, fmt.Errorf("kata: parse roster: %w", err)
	}
	r := &Roster{bySkill: map[string]Creature{}}
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, c := range f.Creatures {
		switch {
		case !idShape.MatchString(c.ID), c.ID != c.SkillID:
			return nil, fmt.Errorf("kata: creature %q must have id == skill_id in [a-z0-9-]", c.ID)
		case ids[c.ID]:
			return nil, fmt.Errorf("kata: duplicate creature %q", c.ID)
		case names[strings.ToLower(c.Name)]:
			return nil, fmt.Errorf("kata: duplicate name %q", c.Name)
		case c.Grade < 1 || c.Grade > 5, c.Regions < 12 || c.Regions > 40, c.Seed <= 0:
			return nil, fmt.Errorf("kata: creature %q has bad grade/regions/seed", c.ID)
		case c.Kind != "math" && c.Kind != "spelling":
			return nil, fmt.Errorf("kata: creature %q kind %q", c.ID, c.Kind)
		case c.Name == "" || c.HintVague == "" || c.HintSpecific == "":
			return nil, fmt.Errorf("kata: creature %q is missing text", c.ID)
		}
		if kind, grade, ok := expectedFor(c.SkillID, cur); !ok || kind != c.Kind || grade != c.Grade {
			return nil, fmt.Errorf("kata: creature %q kind/grade %s/%d do not match its skill", c.ID, c.Kind, c.Grade)
		}
		ids[c.ID] = true
		names[strings.ToLower(c.Name)] = true
		r.creatures = append(r.creatures, c)
		r.bySkill[c.SkillID] = c
	}
	// Coverage: every spelling skill per grade, sight words per grade, and
	// every math op per grade has exactly one creature.
	want := map[string]bool{}
	for _, g := range cur.Grades() {
		for _, s := range cur.Skills(g) {
			want[s.ID] = true
		}
		want[progress.SightSkill(g)] = true
		for _, op := range []string{"addsub", "mul", "div", "frac"} {
			want[progress.MathSkill(op, g)] = true
		}
	}
	for id := range want {
		if _, ok := r.bySkill[id]; !ok {
			return nil, fmt.Errorf("kata: no creature for skill %q", id)
		}
	}
	for id := range r.bySkill {
		if !want[id] {
			return nil, fmt.Errorf("kata: creature %q is bound to an unknown skill", id)
		}
	}
	// Hints must never contain a word from the skill's own bank: the hint
	// points at the skill, it does not give away an answer.
	for _, c := range r.creatures {
		if c.Kind != "spelling" {
			continue
		}
		hint := " " + lettersOnly(c.HintVague+" "+c.HintSpecific) + " "
		var bank []curriculum.Word
		if strings.HasPrefix(c.SkillID, "sight-") {
			bank = cur.SightWords(c.Grade)
		} else {
			for _, w := range cur.Words(c.Grade) {
				if w.Skill == c.SkillID {
					bank = append(bank, w)
				}
			}
		}
		for _, w := range bank {
			if strings.HasPrefix(c.SkillID, "sight-") && hintStopWords[w.Word] {
				continue // an English hint cannot avoid "the"; content words are still checked
			}
			if strings.Contains(hint, " "+w.Word+" ") {
				return nil, fmt.Errorf("kata: hint for %q leaks its word %q", c.ID, w.Word)
			}
		}
	}
	sort.Slice(r.creatures, func(i, j int) bool {
		if r.creatures[i].Grade != r.creatures[j].Grade {
			return r.creatures[i].Grade < r.creatures[j].Grade
		}
		return r.creatures[i].Seed < r.creatures[j].Seed
	})
	return r, nil
}

// hintStopWords are the function words a sight-word hint is allowed to
// contain even though they are themselves sight words. Anything else in the
// band (see, look, little, ...) is still a leak.
var hintStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "in": true, "on": true, "of": true, "to": true, "and": true,
	"or": true, "it": true, "is": true, "are": true, "you": true, "your": true, "that": true,
	"this": true, "with": true, "for": true, "as": true, "at": true, "by": true, "from": true,
}

// expectedFor derives the kind and grade a skill id implies.
func expectedFor(skill string, cur *curriculum.Curriculum) (kind string, grade int, ok bool) {
	var op string
	if n, _ := fmt.Sscanf(skill, "math-%3s-g%d", &op, &grade); n == 2 || strings.HasPrefix(skill, "math-") {
		if n, _ := fmt.Sscanf(skill[strings.LastIndex(skill, "-g")+2:], "%d", &grade); n == 1 {
			return "math", grade, true
		}
		return "", 0, false
	}
	if n, _ := fmt.Sscanf(skill, "sight-g%d", &grade); n == 1 {
		return "spelling", grade, true
	}
	if _, found := cur.Skill(skill); found && len(skill) > 2 && skill[0] == 'g' {
		return "spelling", int(skill[1] - '0'), true
	}
	return "", 0, false
}

// lettersOnly lower-cases and replaces every non-letter with a space so a
// word followed by punctuation still matches as a whole word.
func lettersOnly(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// Creatures returns the roster in display order (by grade, then seed).
func (r *Roster) Creatures() []Creature { return append([]Creature(nil), r.creatures...) }

// BySkill looks up the creature bound to a skill.
func (r *Roster) BySkill(skill string) (Creature, bool) {
	c, ok := r.bySkill[skill]
	return c, ok
}

// Store applies rewards and reads a child's collection.
type Store struct {
	db     *sql.DB
	roster *Roster
}

// New wraps the database and roster.
func New(db *sql.DB, roster *Roster) *Store { return &Store{db: db, roster: roster} }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// Touched is what a round did to one creature, returned to the client so
// the results view can animate the reveal.
type Touched struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Seed       int    `json:"seed"`
	Palette    string `json:"palette"`
	Regions    int    `json:"regions"`
	Fills      int    `json:"fills"`
	FillsAdded int    `json:"fills_added"`
	State      string `json:"state"`
	NewlySeen  bool   `json:"newly_seen"`
	Caught     bool   `json:"caught"`  // crossed into caught in this round
	Evolved    bool   `json:"evolved"` // crossed into evolved in this round
}

// Apply is the progress.Sink: inside the finishing transaction it marks
// every skill in the round as seen, adds fills, and promotes states.
// Idempotency comes from the round itself (a finished round never calls
// this again).
func (s *Store) Apply(ctx context.Context, tx *sql.Tx, child progress.Child, skills []string, reward *progress.Reward, now time.Time) error {
	evolved := map[string]bool{}
	for _, id := range reward.Evolved {
		evolved[id] = true
	}
	seen := map[string]bool{}
	var touched []Touched
	for _, skill := range skills {
		if seen[skill] {
			continue
		}
		seen[skill] = true
		c, ok := s.roster.BySkill(skill)
		if !ok {
			continue // a skill with no creature earns nothing here
		}
		var state string
		var fills int
		err := tx.QueryRowContext(ctx,
			`SELECT state, fills FROM creature_state WHERE child_id = ? AND account_id = ? AND creature_id = ?`,
			child.ChildID, child.AccountID, c.ID).Scan(&state, &fills)
		newly := errors.Is(err, sql.ErrNoRows)
		if err != nil && !newly {
			return err
		}
		if newly {
			state = StateSeen
		}
		t := Touched{ID: c.ID, Name: c.Name, Seed: c.Seed, Palette: c.Palette, Regions: c.Regions, NewlySeen: newly}
		before := fills
		fills = min(c.Regions, fills+reward.FillsBySkill[skill])
		t.FillsAdded = fills - before
		var caughtAt, evolvedAt any
		if state == StateSeen && fills >= c.Regions {
			state, t.Caught, caughtAt = StateCaught, true, ts(now)
		}
		// Mastery recorded earlier (before the creature existed, or on a
		// round this creature was not part of) still evolves it.
		if !evolved[skill] && state != StateEvolved {
			var evolvedOn sql.NullString
			if err := tx.QueryRowContext(ctx,
				`SELECT evolved_on FROM skill_progress WHERE child_id = ? AND account_id = ? AND skill_id = ?`,
				child.ChildID, child.AccountID, skill).Scan(&evolvedOn); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			evolved[skill] = evolvedOn.Valid
		}
		if state != StateEvolved && evolved[skill] {
			if state == StateSeen { // mastery implies enough practice to count as caught
				fills, t.Caught, caughtAt = c.Regions, true, ts(now)
			}
			state, t.Evolved, evolvedAt = StateEvolved, true, ts(now)
		}
		t.Fills, t.State = fills, state
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO creature_state (account_id, child_id, creature_id, state, fills, seen_at, caught_at, evolved_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(child_id, creature_id) DO UPDATE SET
			   state = excluded.state, fills = excluded.fills,
			   caught_at = COALESCE(creature_state.caught_at, excluded.caught_at),
			   evolved_at = COALESCE(creature_state.evolved_at, excluded.evolved_at),
			   updated_at = excluded.updated_at`,
			child.AccountID, child.ChildID, c.ID, state, fills, ts(now), caughtAt, evolvedAt, ts(now)); err != nil {
			return err
		}
		touched = append(touched, t)
	}
	if len(touched) == 0 {
		return nil // nothing practiced, nothing to reveal
	}
	encoded, err := json.Marshal(touched)
	if err != nil {
		return err
	}
	reward.Creatures = encoded
	return nil
}

// Entry is one creature as the index shows it to one child.
type Entry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Grade    int    `json:"grade"`
	SkillID  string `json:"skill_id"`
	Seed     int    `json:"seed"`
	Palette  string `json:"palette"`
	Regions  int    `json:"regions"`
	Fills    int    `json:"fills"`
	State    string `json:"state"`
	Hint     string `json:"hint"`
	Label    string `json:"label"`
	Focus    string `json:"focus"`    // what Train here should set: skill id, "sight-words", or the math op
	Mastered bool   `json:"mastered"` // skill_progress says evolved even if no round has recorded it yet
}

// Index is the whole roster with this child's state merged in. Unknown
// creatures carry the vague hint; anything seen carries the specific one.
type Index struct {
	Entries    []Entry        `json:"entries"`
	ByGrade    map[int][2]int `json:"by_grade"` // [caught or evolved, total]
	ReviewDue  int            `json:"review_due"`
	ChildGrade int            `json:"child_grade"`
}

// Index merges the roster with the child's state. reviewDue is the count
// of words the child keeps missing, computed by the caller.
func (s *Store) Index(ctx context.Context, child progress.Child, prog []progress.SkillProgress, reviewDue int, cur *curriculum.Curriculum) (Index, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT creature_id, state, fills FROM creature_state WHERE child_id = ? AND account_id = ?`,
		child.ChildID, child.AccountID)
	if err != nil {
		return Index{}, err
	}
	defer rows.Close()
	type st struct {
		state string
		fills int
	}
	states := map[string]st{}
	for rows.Next() {
		var id string
		var v st
		if err := rows.Scan(&id, &v.state, &v.fills); err != nil {
			return Index{}, err
		}
		states[id] = v
	}
	if err := rows.Err(); err != nil {
		return Index{}, err
	}
	mastered := map[string]bool{}
	for _, p := range prog {
		if p.Evolved {
			mastered[p.SkillID] = true
		}
	}
	idx := Index{ByGrade: map[int][2]int{}, ReviewDue: reviewDue, ChildGrade: child.Grade}
	for _, c := range s.roster.creatures {
		e := Entry{ID: c.ID, Name: c.Name, Kind: c.Kind, Grade: c.Grade, SkillID: c.SkillID, Seed: c.Seed,
			Palette: c.Palette, Regions: c.Regions, State: StateUnknown, Hint: c.HintVague, Mastered: mastered[c.SkillID]}
		if v, ok := states[c.ID]; ok {
			e.State, e.Fills, e.Hint = v.state, v.fills, c.HintSpecific
		}
		e.Label, e.Focus = labelFor(c, cur)
		totals := idx.ByGrade[c.Grade]
		totals[1]++
		if e.State == StateCaught || e.State == StateEvolved {
			totals[0]++
		}
		idx.ByGrade[c.Grade] = totals
		idx.Entries = append(idx.Entries, e)
	}
	return idx, nil
}

// labelFor derives the human skill label and the setup focus for Train here.
func labelFor(c Creature, cur *curriculum.Curriculum) (label, focus string) {
	if c.Kind == "math" {
		op := strings.TrimSuffix(strings.TrimPrefix(c.SkillID, "math-"), fmt.Sprintf("-g%d", c.Grade))
		labels := map[string]string{"addsub": "Add & Subtract", "mul": "Multiplication", "div": "Division", "frac": "Fractions"}
		return labels[op], op
	}
	if strings.HasPrefix(c.SkillID, "sight-") {
		return "Sight words", curriculum.SightWordSkill
	}
	if s, ok := cur.Skill(c.SkillID); ok {
		return s.Label, s.ID
	}
	return c.SkillID, c.SkillID
}
