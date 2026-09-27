package ost

import (
	"strings"
	"testing"
)

func TestBuildCoversBlueprintAndIsDeterministic(t *testing.T) {
	for grade := 3; grade <= 5; grade++ {
		if len(templates[grade]) == 0 {
			t.Fatalf("grade %d has no templates", grade)
		}
		a, err := Build(grade, 42)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := Build(grade, 42)
		if len(a) != TestLength || len(b) != TestLength {
			t.Fatalf("grade %d built %d items", grade, len(a))
		}
		counts := map[string]int{}
		for i, it := range a {
			counts[it.Category]++
			if it.Prompt != b[i].Prompt || it.Numeric != b[i].Numeric {
				t.Fatalf("grade %d is not deterministic at item %d", grade, i)
			}
			if it.ID == "" || it.Standard == "" || it.DOK < 1 || it.DOK > 3 || it.Explanation == "" {
				t.Fatalf("grade %d item missing metadata: %+v", grade, it)
			}
			// The stored answer must score as correct.
			var ans Answer
			switch it.Type {
			case TypeNumber:
				ans = Answer{Text: it.Numeric}
			default:
				ans = Answer{Choices: it.Answer}
				if len(it.Choices) < 2 || len(it.Answer) == 0 {
					t.Fatalf("choice item without choices: %+v", it)
				}
				seen := map[string]bool{}
				for _, c := range it.Choices {
					if seen[c] {
						t.Fatalf("duplicate choice %q in %s: %v", c, it.Standard, it.Choices)
					}
					seen[c] = true
				}
			}
			if !Check(it, ans) {
				t.Fatalf("key does not score: %+v", it)
			}
		}
		for _, c := range Categories(grade) {
			share := float64(counts[c]) / TestLength
			if share < 0.18 || share > 0.45 {
				t.Fatalf("grade %d category %s has %d items", grade, c, counts[c])
			}
		}
		// A different seed gives a different test.
		c, _ := Build(grade, 7)
		same := 0
		for i := range c {
			if c[i].Prompt == a[i].Prompt {
				same++
			}
		}
		if same > TestLength/2 {
			t.Fatalf("grade %d: seeds 7 and 42 share %d prompts", grade, same)
		}
	}
}

func TestManySeedsProduceValidItems(t *testing.T) {
	for grade := 3; grade <= 5; grade++ {
		for seed := uint64(1); seed <= 300; seed++ {
			items, err := Build(grade, seed)
			if err != nil {
				t.Fatal(err)
			}
			for _, it := range items {
				if strings.Contains(it.Prompt, "%!") || strings.Contains(it.Explanation, "%!") {
					t.Fatalf("format error in %s: %s", it.Standard, it.Prompt)
				}
				if it.Type != TypeNumber {
					seen := map[string]bool{}
					for _, c := range it.Choices {
						c = strings.TrimSpace(c)
						if seen[c] {
							t.Fatalf("grade %d seed %d %s duplicate choice %q", grade, seed, it.Standard, c)
						}
						seen[c] = true
					}
				}
			}
		}
	}
}

func TestCheckAndNumbers(t *testing.T) {
	num := Item{Type: TypeNumber, Numeric: "3/4"}
	for _, ok := range []string{"3/4", " 6/8", "0.75", ".75"} {
		if !Check(num, Answer{Text: ok}) {
			t.Errorf("%q should match 3/4", ok)
		}
	}
	for _, bad := range []string{"", "4/3", "0.7", "three", "7.5e-1", "0,75", "0x1.8p-1", "inf", "+.75"} {
		if Check(num, Answer{Text: bad}) {
			t.Errorf("%q should not match 3/4", bad)
		}
	}
	mixed := Item{Type: TypeNumber, Numeric: "1 3/4"}
	if !Check(mixed, Answer{Text: "7/4"}) || !Check(mixed, Answer{Text: "1.75"}) {
		t.Error("mixed number equivalents should match")
	}
	big := Item{Type: TypeNumber, Numeric: "1234"}
	if !Check(big, Answer{Text: "1,234"}) || Check(big, Answer{Text: "1,2,34"}) || Check(big, Answer{Text: "12,34"}) {
		t.Error("commas must be thousands separators")
	}
	money := Item{Type: TypeNumber, Numeric: "12.50"}
	if !Check(money, Answer{Text: "$12.5"}) || !Check(money, Answer{Text: "12.50"}) {
		t.Error("money formats should match")
	}
	multi := Item{Type: TypeMulti, Answer: []int{0, 2}}
	if !Check(multi, Answer{Choices: []int{2, 0}}) || Check(multi, Answer{Choices: []int{0}}) || Check(multi, Answer{Choices: []int{0, 1, 2}}) {
		t.Error("multi-select must match the exact set")
	}
	single := Item{Type: TypeChoice, Answer: []int{1}}
	if !Check(single, Answer{Choices: []int{1}}) || Check(single, Answer{Choices: []int{1, 2}}) {
		t.Error("single choice must be exactly one")
	}
	for pct, want := range map[int]string{0: "Limited", 39: "Limited", 40: "Basic", 55: "Proficient", 70: "Accomplished", 85: "Advanced", 100: "Advanced"} {
		if got := LevelFor(pct); got != want {
			t.Errorf("LevelFor(%d) = %s, want %s", pct, got, want)
		}
	}
}
