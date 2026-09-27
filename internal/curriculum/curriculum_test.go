package curriculum

import (
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
		"unknown skill":  `{"skills":{"1":[]},"words":{"1":[{"word":"cat","skill":"nope"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{}}`,
		"duplicate word": `{"skills":{"1":[{"id":"a"}]},"words":{"1":[{"word":"cat","skill":"a"},{"word":"cat","skill":"a"}]},"sightWordSkill":{"id":"sight-words"},"sightWords":{}}`,
		"bad grade":      `{"skills":{"9":[]},"words":{},"sightWordSkill":{"id":"sight-words"},"sightWords":{}}`,
		"wrong sight id": `{"skills":{},"words":{},"sightWordSkill":{"id":"sight"},"sightWords":{}}`,
	}
	for name, raw := range cases {
		if _, err := parse([]byte(raw)); err == nil {
			t.Errorf("%s: parse accepted bad data", name)
		}
	}
}

func TestReplay(t *testing.T) {
	cases := []struct {
		name    string
		word    string
		guesses []string
		want    Outcome
	}{
		{"all letters win", "cat", []string{"C", "A", "T"}, Outcome{Won: true, Done: true, Guesses: 3}},
		{"case and whitespace tolerant", "cat", []string{" c", "a", "t "}, Outcome{Won: true, Done: true, Guesses: 3}},
		{"six wrong letters lose", "cat", []string{"B", "D", "E", "F", "G", "H"}, Outcome{Done: true, Mistakes: 6, Guesses: 6}},
		{"five wrong is still open", "cat", []string{"B", "D", "E", "F", "G"}, Outcome{Mistakes: 5, Guesses: 5}},
		{"repeated letter ignored", "cat", []string{"B", "B", "B", "B", "B", "B", "B"}, Outcome{Mistakes: 1, Guesses: 1}},
		{"whole word rescues", "cat", []string{"B", "cat"}, Outcome{Won: true, Done: true, Mistakes: 1, Guesses: 2}},
		{"wrong whole word costs a try", "cat", []string{"cot", "cut", "cap", "can", "car", "cab"}, Outcome{Done: true, Mistakes: 6, Guesses: 6}},
		{"guesses after done ignored", "cat", []string{"C", "A", "T", "Z", "Z", "Z", "Z", "Z", "Z"}, Outcome{Won: true, Done: true, Guesses: 3}},
		{"guesses after loss ignored", "cat", []string{"B", "D", "E", "F", "G", "H", "cat"}, Outcome{Done: true, Mistakes: 6, Guesses: 6}},
		{"blank and non-letter ignored", "cat", []string{"", " ", "1", "C", "A", "T"}, Outcome{Won: true, Done: true, Guesses: 3}},
		{"repeated letters in word", "boat", []string{"B", "O", "A", "T"}, Outcome{Won: true, Done: true, Guesses: 4}},
		{"letter repeats need one guess", "cries", []string{"C", "R", "I", "E", "S"}, Outcome{Won: true, Done: true, Guesses: 5}},
		{"no guesses", "cat", nil, Outcome{}},
	}
	for _, tc := range cases {
		if got := Replay(tc.word, tc.guesses); got != tc.want {
			t.Errorf("%s: Replay = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
