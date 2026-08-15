// Package sheet generates and grades practice worksheets.
package sheet

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand/v2"
	"strings"
	"sync"
	"time"
)

// Subject identifiers. Math is the first dojo; more subjects slot in later.
const SubjectMath = "math"

// Operations within the math subject.
const (
	OpAddSub = "addsub"
	OpMul    = "mul"
	OpDiv    = "div"
	OpFrac   = "frac"
)

// Belt difficulty levels.
const (
	BeltWhite  = "white"
	BeltYellow = "yellow"
	BeltBlack  = "black"
)

var validOps = map[string]bool{OpAddSub: true, OpMul: true, OpDiv: true, OpFrac: true}
var validBelts = map[string]bool{BeltWhite: true, BeltYellow: true, BeltBlack: true}
var validCounts = map[int]bool{10: true, 20: true, 30: true}

// Question is one problem. The answer never leaves the server.
type Question struct {
	Prompt string `json:"prompt"`
	Op     string `json:"op"`
	answer fraction
}

// Sheet is a generated worksheet held server-side until graded.
type Sheet struct {
	ID        string
	Questions []Question
	Belt      string
	CreatedAt time.Time
	graded    bool
}

// Result is the per-question outcome returned after grading.
type Result struct {
	Prompt  string `json:"prompt"`
	Given   string `json:"given"`
	Correct string `json:"correct"`
	Right   bool   `json:"right"`
}

// Store keeps active sheets in memory.
type Store struct {
	mu     sync.Mutex
	sheets map[string]*Sheet
}

func NewStore() *Store { return &Store{sheets: make(map[string]*Sheet)} }

// Generate builds a sheet of count questions drawn evenly from ops at belt difficulty.
func Generate(ops []string, belt string, count int) (*Sheet, error) {
	if len(ops) == 0 {
		return nil, fmt.Errorf("pick at least one operation")
	}
	seen := map[string]bool{}
	for _, op := range ops {
		if !validOps[op] {
			return nil, fmt.Errorf("unknown operation %q", op)
		}
		if seen[op] {
			return nil, fmt.Errorf("duplicate operation %q", op)
		}
		seen[op] = true
	}
	if !validBelts[belt] {
		return nil, fmt.Errorf("unknown belt %q", belt)
	}
	if !validCounts[count] {
		return nil, fmt.Errorf("count must be 10, 20, or 30")
	}

	qs := make([]Question, 0, count)
	for i := 0; i < count; i++ {
		op := ops[i%len(ops)]
		qs = append(qs, genQuestion(op, belt))
	}
	mrand.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })

	id, err := newID()
	if err != nil {
		return nil, err
	}
	return &Sheet{ID: id, Questions: qs, Belt: belt, CreatedAt: time.Now()}, nil
}

// Put stores a sheet, evicting sheets older than an hour.
func (s *Store) Put(sh *Sheet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-time.Hour)
	for id, old := range s.sheets {
		if old.CreatedAt.Before(cutoff) {
			delete(s.sheets, id)
		}
	}
	s.sheets[sh.ID] = sh
}

// Grade scores answers against the stored sheet and removes it.
func (s *Store) Grade(id string, answers []string) ([]Result, error) {
	s.mu.Lock()
	sh, ok := s.sheets[id]
	if ok && len(answers) == len(sh.Questions) {
		delete(s.sheets, id) // consume only on a well-formed submission
	}
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("sheet not found (it may have expired) — start a new one")
	}
	if len(answers) != len(sh.Questions) {
		return nil, fmt.Errorf("expected %d answers, got %d", len(sh.Questions), len(answers))
	}
	results := make([]Result, len(answers))
	for i, q := range sh.Questions {
		given := strings.TrimSpace(answers[i])
		got, err := parseFraction(given)
		results[i] = Result{
			Prompt:  q.Prompt,
			Given:   given,
			Correct: q.answer.String(),
			Right:   err == nil && got.equals(q.answer),
		}
	}
	return results, nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating sheet id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
