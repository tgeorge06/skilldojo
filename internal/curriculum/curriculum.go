// Package curriculum is the server-side view of the spelling curriculum.
// The JSON here is the single source of truth; static/words.js is generated
// from it (make words) so the browser and the server never disagree.
package curriculum

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//go:embed spelling.json
var spellingJSON []byte

// Skill is one teachable spelling focus.
type Skill struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Tip   string `json:"tip"`
}

// Word is one curriculum entry. Rank is set only for sight words.
type Word struct {
	Word     string `json:"word"`
	Skill    string `json:"skill"`
	Clue     string `json:"clue"`
	Sentence string `json:"sentence"`
	Rank     int    `json:"rank,omitempty"`
}

// SightWordSkill is the id of the cross-grade sight-word focus.
const SightWordSkill = "sight-words"

// MaxMistakes ends a Word Rescue round; it mirrors the six lanterns in app.js.
const MaxMistakes = 6

type data struct {
	Skills         map[string][]Skill `json:"skills"`
	Words          map[string][]Word  `json:"words"`
	SightWordSkill Skill              `json:"sightWordSkill"`
	SightWords     map[string][]Word  `json:"sightWords"`
}

var (
	loadOnce sync.Once
	loaded   *Curriculum
	loadErr  error
)

// Curriculum is the parsed, indexed curriculum.
type Curriculum struct {
	skills     map[int][]Skill
	words      map[int][]Word
	sightWords map[int][]Word
	sightSkill Skill
	byID       map[string]Skill
	byWord     map[string]Word
	grades     []int
}

// Load parses the embedded curriculum once and returns it.
func Load() (*Curriculum, error) {
	loadOnce.Do(func() { loaded, loadErr = parse(spellingJSON) })
	return loaded, loadErr
}

// MustLoad is Load for main() and tests.
func MustLoad() *Curriculum {
	c, err := Load()
	if err != nil {
		panic(err)
	}
	return c
}

func parse(raw []byte) (*Curriculum, error) {
	var d data
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("curriculum: parse spelling.json: %w", err)
	}
	if d.SightWordSkill.ID != SightWordSkill {
		return nil, fmt.Errorf("curriculum: sight word skill id is %q, want %q", d.SightWordSkill.ID, SightWordSkill)
	}
	c := &Curriculum{
		skills:     map[int][]Skill{},
		words:      map[int][]Word{},
		sightWords: map[int][]Word{},
		sightSkill: d.SightWordSkill,
		byID:       map[string]Skill{d.SightWordSkill.ID: d.SightWordSkill},
		byWord:     map[string]Word{},
	}
	gradeOf := func(key string) (int, error) {
		g, err := strconv.Atoi(key)
		if err != nil || g < 1 || g > 5 {
			return 0, fmt.Errorf("curriculum: bad grade key %q", key)
		}
		return g, nil
	}
	for key, skills := range d.Skills {
		g, err := gradeOf(key)
		if err != nil {
			return nil, err
		}
		for _, s := range skills {
			if _, dup := c.byID[s.ID]; dup {
				return nil, fmt.Errorf("curriculum: duplicate skill id %q", s.ID)
			}
			c.byID[s.ID] = s
		}
		c.skills[g] = skills
		c.grades = append(c.grades, g)
	}
	sort.Ints(c.grades)
	addWords := func(bank map[string][]Word, dest map[int][]Word, wantSkill string) error {
		// A word may appear in both banks (e.g. "said" is a heart word and a
		// sight word) but never twice within one bank.
		seen := map[string]bool{}
		for key, words := range bank {
			g, err := gradeOf(key)
			if err != nil {
				return err
			}
			for i := range words {
				w := words[i]
				if wantSkill != "" {
					w.Skill = wantSkill
					words[i].Skill = wantSkill
				}
				if _, ok := c.byID[w.Skill]; !ok {
					return fmt.Errorf("curriculum: word %q has unknown skill %q", w.Word, w.Skill)
				}
				if seen[w.Word] {
					return fmt.Errorf("curriculum: duplicate word %q", w.Word)
				}
				seen[w.Word] = true
				if _, ok := c.byWord[w.Word]; !ok {
					c.byWord[w.Word] = w
				}
			}
			dest[g] = words
		}
		return nil
	}
	if err := addWords(d.Words, c.words, ""); err != nil {
		return nil, err
	}
	if err := addWords(d.SightWords, c.sightWords, SightWordSkill); err != nil {
		return nil, err
	}
	return c, nil
}

// Grades lists the grades that have skills, ascending.
func (c *Curriculum) Grades() []int { return append([]int(nil), c.grades...) }

// Skills returns the pattern skills for a grade (not including sight words).
func (c *Curriculum) Skills(grade int) []Skill { return append([]Skill(nil), c.skills[grade]...) }

// Skill looks up any skill, including the sight-word skill.
func (c *Curriculum) Skill(id string) (Skill, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// Words returns the pattern words for a grade.
func (c *Curriculum) Words(grade int) []Word { return append([]Word(nil), c.words[grade]...) }

// SightWords returns the sight-word band for a grade.
func (c *Curriculum) SightWords(grade int) []Word { return append([]Word(nil), c.sightWords[grade]...) }

// Word looks up a word from either bank, case-insensitively.
func (c *Curriculum) Word(word string) (Word, bool) {
	w, ok := c.byWord[strings.ToLower(strings.TrimSpace(word))]
	return w, ok
}

// AllWords returns every entry from both banks; order is by grade then bank order.
func (c *Curriculum) AllWords() []Word {
	var out []Word
	for _, g := range c.grades {
		out = append(out, c.words[g]...)
	}
	for _, g := range c.grades {
		out = append(out, c.sightWords[g]...)
	}
	return out
}

// Outcome is the server-side verdict of a replayed Word Rescue round.
type Outcome struct {
	Won      bool
	Done     bool
	Mistakes int
	// Guesses is the number of guesses that changed state; ignored guesses
	// (repeats, blanks, anything after the round ended) are not counted.
	Guesses int
}

// Replay applies a client's ordered guesses to a word exactly as app.js does:
// a single letter reveals or costs a try; anything longer is a whole-word
// guess that either rescues the word or costs a try; six tries ends the
// round. Repeated letters, blanks, and guesses after the round ends are
// ignored, so a replay can never be steered into a better result than the
// client actually earned.
func Replay(word string, guesses []string) Outcome {
	target := strings.ToUpper(strings.TrimSpace(word))
	var out Outcome
	revealed := map[rune]bool{}
	need := map[rune]bool{}
	for _, r := range target {
		need[r] = true
	}
	allRevealed := func() bool {
		for r := range need {
			if !revealed[r] {
				return false
			}
		}
		return true
	}
	for _, raw := range guesses {
		if out.Done {
			break
		}
		g := strings.ToUpper(strings.TrimSpace(raw))
		runes := []rune(g)
		switch {
		case len(runes) == 0:
			continue
		case len(runes) == 1:
			r := runes[0]
			if r < 'A' || r > 'Z' || revealed[r] {
				continue
			}
			// Any letter is recorded as guessed, matching guessedLetters.
			revealed[r] = true
			out.Guesses++
			if need[r] {
				if allRevealed() {
					out.Won, out.Done = true, true
				}
				continue
			}
			out.Mistakes++
		default:
			out.Guesses++
			if g == target {
				out.Won, out.Done = true, true
				continue
			}
			out.Mistakes++
		}
		if out.Mistakes >= MaxMistakes {
			out.Done = true
		}
	}
	return out
}
