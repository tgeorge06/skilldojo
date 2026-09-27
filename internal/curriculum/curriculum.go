// Package curriculum is the server-side view of the spelling curriculum.
// The JSON here is the single source of truth; static/words.js is generated
// from it (make words) so the browser and the server never disagree.
package curriculum

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
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

// wordShape is the invariant every curriculum word satisfies; the browser
// compares guesses against these bytes exactly.
var wordShape = regexp.MustCompile(`^[a-z]+$`)

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
	// Grade keys must be exactly "1".."5": the generator and the browser index
	// by the canonical string, so "01" would silently split a grade.
	gradeOf := func(key string) (int, error) {
		g, err := strconv.Atoi(key)
		if err != nil || g < 1 || g > 5 || strconv.Itoa(g) != key {
			return 0, fmt.Errorf("curriculum: bad grade key %q", key)
		}
		return g, nil
	}
	sameGrades := func(name string, bank map[string][]Word) error {
		if len(bank) != len(d.Skills) {
			return fmt.Errorf("curriculum: %s covers %d grades, skills cover %d", name, len(bank), len(d.Skills))
		}
		for key := range bank {
			if _, ok := d.Skills[key]; !ok {
				return fmt.Errorf("curriculum: %s has grade %q with no skills", name, key)
			}
		}
		return nil
	}
	if err := sameGrades("words", d.Words); err != nil {
		return nil, err
	}
	if err := sameGrades("sightWords", d.SightWords); err != nil {
		return nil, err
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
				if !wordShape.MatchString(w.Word) {
					return fmt.Errorf("curriculum: word %q must be lowercase a-z with no spaces", w.Word)
				}
				if wantSkill == SightWordSkill && w.Rank <= 0 {
					return fmt.Errorf("curriculum: sight word %q needs a positive rank", w.Word)
				}
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
	Won      bool `json:"won"`
	Done     bool `json:"done"`
	Mistakes int  `json:"mistakes"`
}

// Guess is one event from the client, tagged by how it was made: a letter
// from the board or keyboard, or a whole-word submission from the text box.
// The kinds have different rules, so the client must say which it was.
type Guess struct {
	Kind  string `json:"kind"` // "letter" or "word"
	Value string `json:"value"`
}

const (
	GuessLetter = "letter"
	GuessWord   = "word"
)

// Replay applies a client's ordered guesses to a word with the same rules
// as app.js: a letter reveals or costs a try and is ignored if already
// guessed; a whole-word submission rescues the word or costs a try every
// time, even when repeated; six tries end the round; events after the round
// ends are ignored.
//
// Where the browser is more permissive (Unicode case folding, exotic
// whitespace), the server is deliberately stricter: a word guess counts as
// a rescue only if, after trimming ASCII whitespace, it is byte-for-byte the
// curriculum word. Anything else is charged as a wrong guess. That means a
// replay can be harsher than the browser in bizarre inputs but never
// kinder, so it cannot be steered into a win the client did not earn.
func Replay(word string, guesses []Guess) Outcome {
	var out Outcome
	if !wordShape.MatchString(word) {
		return out
	}
	target := strings.ToUpper(word)
	revealed := map[byte]bool{}
	seen := map[string]bool{}
	allRevealed := func() bool {
		for i := 0; i < len(target); i++ {
			if !revealed[target[i]] {
				return false
			}
		}
		return true
	}
	for _, g := range guesses {
		if out.Done {
			break
		}
		switch g.Kind {
		case GuessLetter:
			// guessLetter ignores a value it has already seen and charges any
			// value not in the word. A malformed value (lowercase, multi-char)
			// can never match, so it is charged too: ignoring it would let a
			// client relabel its wrong guesses and replay to a win.
			if g.Value == "" || seen[g.Value] {
				continue
			}
			seen[g.Value] = true
			if len(g.Value) == 1 && g.Value[0] >= 'A' && g.Value[0] <= 'Z' && strings.IndexByte(target, g.Value[0]) >= 0 {
				revealed[g.Value[0]] = true
				if allRevealed() {
					out.Won, out.Done = true, true
				}
				continue
			}
			out.Mistakes++
		case GuessWord:
			v := strings.Trim(g.Value, " \t\r\n")
			if v == "" {
				continue // guessWholeWord ignores blank submissions
			}
			if v == word {
				out.Won, out.Done = true, true
				continue
			}
			out.Mistakes++
		default:
			continue
		}
		if out.Mistakes >= MaxMistakes {
			out.Done = true
		}
	}
	return out
}
