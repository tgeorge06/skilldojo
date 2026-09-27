// Package ost builds Ohio's State Tests style mathematics practice tests
// for grades 3 to 5. Items are generated from standard-aligned templates,
// so every attempt is a fresh test; each item is tagged with its reporting
// category, learning standard, and depth of knowledge, which is what the
// parent report groups results by.
//
// Reporting categories follow Ohio's published test blueprints. Items are
// original and written in the style of the test; they are practice, not
// the test, and the performance-level estimate is a rough guide only.
package ost

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	TypeChoice = "choice" // one correct choice
	TypeMulti  = "multi"  // choose all that apply
	TypeNumber = "number" // typed numeric answer (integer, decimal, or fraction)
)

// Item is one generated question. Answer is kept server-side; the client
// gets Public().
type Item struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Standard    string   `json:"standard"`
	DOK         int      `json:"dok"`
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Choices     []string `json:"choices,omitempty"`
	Answer      []int    `json:"answer,omitempty"`  // choice indexes (one for choice, several for multi)
	Numeric     string   `json:"numeric,omitempty"` // canonical numeric answer for TypeNumber
	Explanation string   `json:"explanation"`
}

// PublicItem is what the child sees.
type PublicItem struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Type     string   `json:"type"`
	Prompt   string   `json:"prompt"`
	Choices  []string `json:"choices,omitempty"`
}

// Public strips the answer.
func (it Item) Public() PublicItem {
	return PublicItem{ID: it.ID, Category: it.Category, Type: it.Type, Prompt: it.Prompt, Choices: it.Choices}
}

// Template makes items for one standard.
type Template struct {
	Category string
	Standard string
	DOK      int
	Gen      func(r *rand.Rand) Item
}

// Category weights per grade follow the blueprints' approximate portions.
type categoryWeight struct {
	Name   string
	Weight float64
}

var blueprint = map[int][]categoryWeight{
	3: {{"Multiplication and Division", 0.28}, {"Number and Operations", 0.24}, {"Fractions", 0.24}, {"Geometry", 0.24}},
	4: {{"Multiplication and Division", 0.38}, {"Fractions", 0.38}, {"Geometry", 0.24}},
	5: {{"Fractions", 0.38}, {"Decimals", 0.38}, {"Geometry", 0.24}},
}

// Categories lists a grade's reporting categories in blueprint order.
func Categories(grade int) []string {
	var out []string
	for _, c := range blueprint[grade] {
		out = append(out, c.Name)
	}
	return out
}

// TestLength is the number of items on a practice test.
const TestLength = 40

var templates = map[int][]Template{}

func register(grade int, ts ...Template) { templates[grade] = append(templates[grade], ts...) }

// Build assembles a test for a grade from a seed. The same seed yields the
// same test, so an attempt can be regenerated from its seed if needed.
func Build(grade int, seed uint64) ([]Item, error) {
	weights, ok := blueprint[grade]
	if !ok {
		return nil, errors.New("ost: practice tests cover grades 3 to 5")
	}
	pool := templates[grade]
	if len(pool) == 0 {
		return nil, fmt.Errorf("ost: no templates for grade %d", grade)
	}
	r := rand.New(rand.NewPCG(seed, 0x057))
	byCat := map[string][]Template{}
	for _, t := range pool {
		byCat[t.Category] = append(byCat[t.Category], t)
	}
	// Item counts per category: largest-remainder rounding of the weights.
	counts := make([]int, len(weights))
	total := 0
	rem := make([]float64, len(weights))
	for i, w := range weights {
		exact := w.Weight * TestLength
		counts[i] = int(math.Floor(exact))
		rem[i] = exact - float64(counts[i])
		total += counts[i]
	}
	for total < TestLength {
		best := 0
		for i := range rem {
			if rem[i] > rem[best] {
				best = i
			}
		}
		counts[best]++
		rem[best] = -1
		total++
	}
	var items []Item
	for i, w := range weights {
		ts := byCat[w.Name]
		if len(ts) == 0 {
			return nil, fmt.Errorf("ost: no templates for %s grade %d", w.Name, grade)
		}
		// Rotate through the category's templates so a test covers many
		// standards before repeating one.
		order := r.Perm(len(ts))
		for k := 0; k < counts[i]; k++ {
			t := ts[order[k%len(order)]]
			it := t.Gen(r)
			it.Category, it.Standard, it.DOK = w.Name, t.Standard, t.DOK
			it.ID = fmt.Sprintf("%d-%s-%d", grade, strings.ReplaceAll(strings.ToLower(t.Standard), ".", ""), len(items)+1)
			items = append(items, it)
		}
	}
	r.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	for i := range items {
		items[i].ID = fmt.Sprintf("q%02d", i+1)
	}
	return items, nil
}

// Answer is what the child submitted for one item.
type Answer struct {
	Choices []int  `json:"choices,omitempty"`
	Text    string `json:"text,omitempty"`
}

// Check scores one answer. Multi-select earns the point only when the set
// matches exactly, as on the test.
func Check(it Item, a Answer) bool {
	switch it.Type {
	case TypeChoice:
		return len(a.Choices) == 1 && len(it.Answer) == 1 && a.Choices[0] == it.Answer[0]
	case TypeMulti:
		if len(a.Choices) != len(it.Answer) {
			return false
		}
		want := append([]int(nil), it.Answer...)
		got := append([]int(nil), a.Choices...)
		sort.Ints(want)
		sort.Ints(got)
		for i := range want {
			if want[i] != got[i] {
				return false
			}
		}
		return true
	case TypeNumber:
		return sameNumber(a.Text, it.Numeric)
	}
	return false
}

// sameNumber compares a typed value to the key: integers, decimals, and
// fractions (including equivalent ones) all count.
func sameNumber(given, key string) bool {
	g, ok1 := parseNumber(given)
	k, ok2 := parseNumber(key)
	return ok1 && ok2 && math.Abs(g-k) < 1e-6
}

var thousands = regexp.MustCompile(`^-?\d{1,3}(,\d{3})+(\.\d+)?$`)

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	if s == "" || strings.ContainsAny(s, "eE") {
		return 0, false // no exponents; "1e3" is not a grade-school answer
	}
	if strings.Contains(s, ",") {
		if !thousands.MatchString(s) {
			return 0, false // "1,2,3" is not a number
		}
		s = strings.ReplaceAll(s, ",", "")
	}
	if strings.Contains(s, "/") {
		// mixed number "1 3/4" or fraction "3/4"
		whole := 0.0
		if parts := strings.Fields(s); len(parts) == 2 {
			w, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return 0, false
			}
			whole, s = w, parts[1]
		}
		num, den, ok := strings.Cut(s, "/")
		if !ok {
			return 0, false
		}
		n, err1 := strconv.ParseFloat(num, 64)
		d, err2 := strconv.ParseFloat(den, 64)
		if err1 != nil || err2 != nil || d == 0 {
			return 0, false
		}
		if whole < 0 {
			return whole - n/d, true
		}
		return whole + n/d, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// Levels are an approximate mapping from percent correct to Ohio's five
// performance levels. The real test uses scaled scores; this is a guide.
type Level struct {
	Name string
	Min  int // percent
}

var Levels = []Level{{"Limited", 0}, {"Basic", 40}, {"Proficient", 55}, {"Accomplished", 70}, {"Advanced", 85}}

// LevelFor returns the estimated level name for a percent.
func LevelFor(percent int) string {
	name := Levels[0].Name
	for _, l := range Levels {
		if percent >= l.Min {
			name = l.Name
		}
	}
	return name
}

// ---- generation helpers shared by the grade files ----

var names = []string{"Maya", "Leo", "Priya", "Owen", "Zara", "Eli", "Nora", "Sam", "Ava", "Jonah", "Mia", "Theo", "Ivy", "Kai", "Ruby", "Mateo"}
var things = []string{"stickers", "marbles", "books", "apples", "crayons", "seashells", "tickets", "beads", "cookies", "pencils", "buttons", "cards"}
var containers = []string{"boxes", "bags", "jars", "baskets", "shelves", "packs", "rows", "trays"}

func pick[T any](r *rand.Rand, xs []T) T   { return xs[r.IntN(len(xs))] }
func between(r *rand.Rand, lo, hi int) int { return lo + r.IntN(hi-lo+1) }

// choices builds a choice item with the correct answer placed at a random
// index. Distractors are deduplicated against the answer and each other
// and the first three distinct ones are used, so a template whose
// distractors can collide at edge values may pass extra candidates.
func choices(r *rand.Rand, prompt, correct string, distractors []string, explanation string) Item {
	seen := map[string]bool{correct: true}
	var ds []string
	for _, d := range distractors {
		if !seen[d] {
			seen[d] = true
			ds = append(ds, d)
		}
	}
	if len(ds) < 3 {
		// Templates must supply three distinct distractors; the tests
		// build many seeds per grade to prove they do.
		panic(fmt.Sprintf("ost: item %q has only %d distinct distractors", prompt, len(ds)))
	}
	ds = ds[:3]
	pos := r.IntN(4)
	opts := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		if i == pos {
			opts = append(opts, correct)
		} else {
			opts = append(opts, ds[0])
			ds = ds[1:]
		}
	}
	return Item{Type: TypeChoice, Prompt: prompt, Choices: opts, Answer: []int{pos}, Explanation: explanation}
}

func numeric(prompt, answer, explanation string) Item {
	return Item{Type: TypeNumber, Prompt: prompt, Numeric: answer, Explanation: explanation}
}

// multi builds a choose-all item from options with a set of correct indexes.
func multi(prompt string, opts []string, correct []int, explanation string) Item {
	return Item{Type: TypeMulti, Prompt: prompt + " Choose all that apply.", Choices: opts, Answer: correct, Explanation: explanation}
}

func itoa(n int) string    { return strconv.Itoa(n) }
func frac(n, d int) string { return fmt.Sprintf("%d/%d", n, d) }
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
func simplify(n, d int) string {
	g := gcd(n, d)
	if g == 0 {
		g = 1
	}
	n, d = n/g, d/g
	if d == 1 {
		return itoa(n)
	}
	return frac(n, d)
}
func dec(v float64, places int) string { return strconv.FormatFloat(v, 'f', places, 64) }
