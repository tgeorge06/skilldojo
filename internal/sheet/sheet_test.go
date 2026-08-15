package sheet

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseFraction(t *testing.T) {
	cases := []struct {
		in   string
		want fraction
		ok   bool
	}{
		{"12", whole(12), true},
		{" 7 ", whole(7), true},
		{"3/4", frac(3, 4), true},
		{"2/4", frac(1, 2), true}, // unreduced input reduces
		{"6/3", whole(2), true},
		{"", fraction{}, false},
		{"1/0", fraction{}, false},
		{"abc", fraction{}, false},
		{"1/x", fraction{}, false},
	}
	for _, c := range cases {
		got, err := parseFraction(c.in)
		if c.ok != (err == nil) {
			t.Fatalf("parseFraction(%q) err=%v, want ok=%v", c.in, err, c.ok)
		}
		if c.ok && !got.equals(c.want) {
			t.Errorf("parseFraction(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFractionString(t *testing.T) {
	if s := frac(2, 4).String(); s != "1/2" {
		t.Errorf("frac(2,4) = %q, want 1/2", s)
	}
	if s := whole(9).String(); s != "9" {
		t.Errorf("whole(9) = %q, want 9", s)
	}
}

func TestGenerateValidation(t *testing.T) {
	if _, err := Generate(nil, BeltWhite, 10); err == nil {
		t.Error("empty ops should fail")
	}
	if _, err := Generate([]string{"bogus"}, BeltWhite, 10); err == nil {
		t.Error("bad op should fail")
	}
	if _, err := Generate([]string{OpMul, OpMul}, BeltWhite, 10); err == nil {
		t.Error("duplicate ops should fail")
	}
	if _, err := Generate([]string{OpMul}, "rainbow", 10); err == nil {
		t.Error("bad belt should fail")
	}
	if _, err := Generate([]string{OpMul}, BeltWhite, 7); err == nil {
		t.Error("bad count should fail")
	}
}

// Every generated question, at every belt, must have a well-formed prompt and a
// non-negative answer; division/multiplication answers must be whole.
func TestGeneratedQuestionsSound(t *testing.T) {
	ops := []string{OpAddSub, OpMul, OpDiv, OpFrac}
	for _, belt := range []string{BeltWhite, BeltYellow, BeltBlack} {
		for i := 0; i < 200; i++ {
			sh, err := Generate(ops, belt, 20)
			if err != nil {
				t.Fatalf("Generate(%s): %v", belt, err)
			}
			if len(sh.Questions) != 20 {
				t.Fatalf("got %d questions, want 20", len(sh.Questions))
			}
			for _, q := range sh.Questions {
				if q.Prompt == "" {
					t.Fatalf("empty prompt (%s)", belt)
				}
				if q.answer.num < 0 || q.answer.den <= 0 {
					t.Fatalf("bad answer %v for %q (%s)", q.answer, q.Prompt, belt)
				}
				if (q.Op == OpMul || q.Op == OpDiv || q.Op == OpAddSub) && q.answer.den != 1 {
					t.Fatalf("non-whole answer %v for %q", q.answer, q.Prompt)
				}
			}
		}
	}
}

// Verify prompts actually evaluate to the stored answer.
func TestPromptMatchesAnswer(t *testing.T) {
	for i := 0; i < 500; i++ {
		sh, err := Generate([]string{OpAddSub, OpMul, OpDiv}, BeltBlack, 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, q := range sh.Questions {
			parts := strings.Fields(q.Prompt)
			if len(parts) != 3 {
				t.Fatalf("unexpected prompt shape %q", q.Prompt)
			}
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[2])
			var want int
			switch parts[1] {
			case "+":
				want = a + b
			case "−":
				want = a - b
			case "×":
				want = a * b
			case "÷":
				want = a / b
				if a%b != 0 {
					t.Fatalf("%q does not divide evenly", q.Prompt)
				}
			default:
				t.Fatalf("unknown operator in %q", q.Prompt)
			}
			if !q.answer.equals(whole(want)) {
				t.Fatalf("%q: stored answer %v, want %d", q.Prompt, q.answer, want)
			}
		}
	}
}

func TestStoreGrade(t *testing.T) {
	st := NewStore()
	sh, err := Generate([]string{OpMul}, BeltWhite, 10)
	if err != nil {
		t.Fatal(err)
	}
	st.Put(sh)

	answers := make([]string, len(sh.Questions))
	for i, q := range sh.Questions {
		if i%2 == 0 {
			answers[i] = q.answer.String() // right
		} else {
			answers[i] = "999999" // wrong
		}
	}
	results, err := st.Grade(sh.ID, answers)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range results {
		if wantRight := i%2 == 0; r.Right != wantRight {
			t.Errorf("q%d right=%v, want %v (given %q, correct %q)", i, r.Right, wantRight, r.Given, r.Correct)
		}
	}

	if _, err := st.Grade(sh.ID, answers); err == nil {
		t.Error("second grade of same sheet should fail (sheet consumed)")
	}
	if _, err := st.Grade("nope", nil); err == nil {
		t.Error("unknown sheet should fail")
	}

	sh2, _ := Generate([]string{OpMul}, BeltWhite, 10)
	st.Put(sh2)
	if _, err := st.Grade(sh2.ID, []string{"1"}); err == nil {
		t.Error("wrong answer count should fail")
	}
	// A malformed submission must not consume the sheet.
	good := make([]string, len(sh2.Questions))
	if _, err := st.Grade(sh2.ID, good); err != nil {
		t.Errorf("sheet should survive a malformed grade attempt: %v", err)
	}
}

func TestGradeAcceptsEquivalentFractions(t *testing.T) {
	st := NewStore()
	sh := &Sheet{ID: "t1", CreatedAt: time.Now(), Questions: []Question{{Prompt: "1/4 + 1/4", Op: OpFrac, answer: frac(1, 2)}}}
	st.Put(sh)
	results, err := st.Grade("t1", []string{"2/4"})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Right {
		t.Error("2/4 should grade equal to 1/2")
	}
}
