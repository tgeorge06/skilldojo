package curriculum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestLoadShape(t *testing.T) {
	c := MustLoad()
	if got := c.Grades(); len(got) != 5 || got[0] != 1 || got[4] != 5 {
		t.Fatalf("grades = %v, want 1..5", got)
	}
	var skills, words, sight int
	for _, g := range c.Grades() {
		skills += len(c.Skills(g))
		words += len(c.Words(g))
		sight += len(c.SightWords(g))
		if len(c.SightWords(g)) != 10 {
			t.Errorf("grade %d has %d sight words, want 10", g, len(c.SightWords(g)))
		}
		for _, s := range c.Skills(g) {
			n := 0
			for _, w := range c.Words(g) {
				if w.Skill == s.ID {
					n++
				}
			}
			if n < 5 {
				t.Errorf("skill %s has %d words, want >= 5", s.ID, n)
			}
		}
	}
	if skills != 26 || words != 130 || sight != 50 {
		t.Fatalf("skills/words/sight = %d/%d/%d, want 26/130/50", skills, words, sight)
	}
	if _, ok := c.Skill(SightWordSkill); !ok {
		t.Fatal("sight-word skill missing")
	}
	if w, ok := c.Word(" Cat "); !ok || w.Skill != "g1-short-vowels" {
		t.Fatalf("Word(cat) = %+v, %v", w, ok)
	}
	all := c.AllWords()
	unique := map[string]bool{}
	for _, w := range all {
		unique[w.Word] = true
	}
	// Four words are both heart words and sight words; the audio set dedupes them.
	if len(all) != 180 || len(unique) != 176 {
		t.Fatalf("AllWords = %d entries / %d unique, want 180 / 176", len(all), len(unique))
	}
	if w, ok := c.Word("said"); !ok || w.Skill == SightWordSkill {
		t.Fatalf("Word(said) should resolve to the pattern-bank entry, got %+v", w)
	}
}

// Mirrors the JS curriculum test: every sentence contains its word exactly
// once as a whole word, because blankedSentence masks only the first match.
func TestSentencesContainWordExactlyOnce(t *testing.T) {
	for _, w := range MustLoad().AllWords() {
		if !regexp.MustCompile(`^[a-z]+$`).MatchString(w.Word) {
			t.Errorf("word %q is not lowercase a-z", w.Word)
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w.Word) + `\b`)
		if n := len(re.FindAllStringIndex(w.Sentence, -1)); n != 1 {
			t.Errorf("%q appears %d times in %q, want 1", w.Word, n, w.Sentence)
		}
	}
}

func TestParseRejectsBadData(t *testing.T) {
	cases := map[string]string{
		"unknown skill":        `{"skills":{"1":[]},"words":{"1":[{"word":"cat","skill":"nope"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"duplicate word":       `{"skills":{"1":[{"id":"a"}]},"words":{"1":[{"word":"cat","skill":"a"},{"word":"cat","skill":"a"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"bad grade":            `{"skills":{"9":[]},"words":{"9":[]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"9":[]}}`,
		"noncanonical grade":   `{"skills":{"01":[]},"words":{"01":[]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"01":[]}}`,
		"grade sets differ":    `{"skills":{"1":[]},"words":{"1":[],"2":[]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"wrong sight id":       `{"skills":{},"words":{},"sightWordSkill":{"id":"sight"},"sightWords":{}}`,
		"padded word":          `{"skills":{"1":[{"id":"a"}]},"words":{"1":[{"word":" cat ","skill":"a"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"uppercase word":       `{"skills":{"1":[{"id":"a"}]},"words":{"1":[{"word":"Cat","skill":"a"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"empty word":           `{"skills":{"1":[{"id":"a"}]},"words":{"1":[{"word":"","skill":"a"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[]}}`,
		"sight word with dash": `{"skills":{"1":[]},"words":{"1":[]},"sightWordSkill":{"id":"sight-words"},"sightWords":{"1":[{"word":"don't","rank":1}]}}`,
	}
	for name, raw := range cases {
		if _, err := parse([]byte(raw)); err == nil {
			t.Errorf("%s: parse accepted bad data", name)
		}
	}
}

// replayFixture mirrors tests/fixtures/replay.json, which the JS suite also
// drives through the real game component so both runtimes agree.
type replayFixture struct {
	Name    string  `json:"name"`
	Word    string  `json:"word"`
	Guesses []Guess `json:"guesses"`
	Want    Outcome `json:"want"`
}

func TestReplayMatchesSharedFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "replay.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []replayFixture
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 10 {
		t.Fatalf("only %d fixtures", len(cases))
	}
	for _, tc := range cases {
		if got := Replay(tc.Word, tc.Guesses); got != tc.Want {
			t.Errorf("%s: Replay = %+v, want %+v", tc.Name, got, tc.Want)
		}
	}
}

// Server-only strictness: inputs the browser might accept through Unicode
// case folding or exotic whitespace are charged, never rewarded.
func TestReplayIsNeverKinderThanTheBrowser(t *testing.T) {
	for _, v := range []string{"\u212Aite", "kite\u0085", "\u00a0kite", "KITE", "Kite"} {
		got := Replay("kite", []Guess{{GuessWord, v}})
		if got.Won || got.Mistakes != 1 {
			t.Errorf("word guess %q: %+v, want one mistake and no win", v, got)
		}
	}
	for _, v := range []string{"k", "kk", "", "1", "\u212A"} {
		got := Replay("kite", []Guess{{GuessLetter, v}})
		if got.Won || got.Mistakes != 0 {
			t.Errorf("letter guess %q: %+v, want ignored", v, got)
		}
	}
	if got := Replay(" cat ", []Guess{{GuessWord, "cat"}}); got.Won {
		t.Error("malformed target must never be winnable")
	}
}
