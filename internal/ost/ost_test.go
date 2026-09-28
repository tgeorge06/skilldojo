package ost

import (
	"regexp"
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

func TestFiguresAndGlossary(t *testing.T) {
	kinds := map[string]bool{"rect": true, "grid": true, "parts": true, "numberline": true, "bars": true, "lineplot": true}
	figures := 0
	for grade := 3; grade <= 5; grade++ {
		for seed := uint64(1); seed <= 40; seed++ {
			items, _ := Build(grade, seed)
			for _, it := range items {
				if it.Figure == nil {
					continue
				}
				figures++
				f := it.Figure
				if !kinds[f.Kind] || f.A < 1 || f.B < 0 || f.A > 60 || f.B > 60 || (f.Kind == "bars" && len(f.Names) != len(f.Values)) {
					t.Fatalf("bad figure on %s: %+v", it.Standard, f)
				}
				if (f.Kind == "parts" || f.Kind == "numberline") && f.B > f.A {
					t.Fatalf("shaded/point past the end on %s: %+v", it.Standard, f)
				}
				if it.Public().Figure == nil {
					t.Fatal("figures must reach the child")
				}
				if f.Alt == "" || !strings.Contains(f.Alt, "picture") {
					t.Fatalf("figure without a spoken description on %s: %+v", it.Standard, f)
				}
				if f.Kind == "rect" && f.LabelA != "?" && f.LabelB != "?" && f.A < f.B {
					t.Fatalf("rect drawn with the short side long on %s: %+v", it.Standard, f)
				}
			}
		}
	}
	if figures == 0 {
		t.Fatal("no figures generated")
	}
	// A picture with a "?" side never carries the keyed number.
	hidden := Item{Type: TypeNumber, Numeric: "4", Figure: &Figure{Kind: "rect", A: 9, B: 4, LabelA: "9 ft", LabelB: "?"}}
	if pub := hidden.Public().Figure; pub.B == 4 || pub.A != 9 || pub.LabelB != "?" {
		t.Fatalf("public figure leaks the unknown side: %+v", pub)
	}
	if hidden.Figure.B != 4 {
		t.Fatal("the stored figure must keep the real value for the report")
	}
	for word, def := range Terms {
		if word != strings.ToLower(word) || strings.TrimSpace(def) == "" || len(def) > 90 {
			t.Fatalf("glossary entry %q: keys are lowercase, definitions short and present", word)
		}
	}
	// The words a nine-year-old stumbled on are in the glossary.
	for _, w := range []string{"perimeter", "rectangle", "area", "equivalent", "quadrilateral"} {
		if _, ok := Terms[w]; !ok {
			t.Fatalf("glossary is missing %q", w)
		}
	}
}

// Ohio's grade 3 blueprint and content limits: every listed standard has a
// template, fractions stay on the allowed denominators, and a test never
// repeats a question.
func TestGrade3FollowsOhioBlueprint(t *testing.T) {
	want := map[string]string{
		"3.OA.1": "Multiplication and Division", "3.OA.2": "Multiplication and Division", "3.OA.3": "Multiplication and Division",
		"3.OA.4": "Multiplication and Division", "3.OA.5": "Multiplication and Division", "3.OA.6": "Multiplication and Division",
		"3.OA.7": "Multiplication and Division", "3.OA.8": "Multiplication and Division", "3.OA.9": "Multiplication and Division",
		"3.NBT.3": "Multiplication and Division",
		"3.NBT.1": "Number and Operations", "3.NBT.2": "Number and Operations", "3.MD.1": "Number and Operations",
		"3.MD.2": "Number and Operations", "3.MD.3": "Number and Operations",
		"3.MD.5": "Geometry", "3.MD.6": "Geometry", "3.MD.7": "Geometry", "3.MD.8": "Geometry", "3.G.1": "Geometry", "3.G.2": "Geometry",
		"3.NF.1": "Fractions", "3.NF.2": "Fractions", "3.NF.3": "Fractions", "3.MD.4": "Fractions",
	}
	have := map[string]string{}
	for _, tp := range templates[3] {
		if c, ok := want[tp.Standard]; !ok {
			t.Errorf("template for %s is not on the grade 3 blueprint", tp.Standard)
		} else if c != tp.Category {
			t.Errorf("%s is filed under %q; the blueprint puts it in %q", tp.Standard, tp.Category, c)
		}
		have[tp.Standard] = tp.Category
	}
	for std := range want {
		if _, ok := have[std]; !ok {
			t.Errorf("no template covers %s", std)
		}
	}
	allowed := map[string]bool{"2": true, "3": true, "4": true, "6": true, "8": true, "1": true}
	fracRE := regexp.MustCompile(`\b\d+/(\d+)\b`)
	for seed := uint64(1); seed <= 200; seed++ {
		items, _ := Build(3, seed)
		seen := map[string]bool{}
		for _, it := range items {
			key := it.Prompt + "|" + strings.Join(it.Choices, "|") + "|" + figureKey(it.Figure)
			if seen[key] {
				t.Fatalf("seed %d repeats a question: %s", seed, it.Prompt)
			}
			seen[key] = true
			if it.Category != "Fractions" && it.Standard != "3.G.2" {
				continue
			}
			for _, text := range append([]string{it.Prompt, it.Numeric}, it.Choices...) {
				for _, m := range fracRE.FindAllStringSubmatch(text, -1) {
					if !allowed[m[1]] {
						t.Fatalf("seed %d %s uses denominator %s (%s)", seed, it.Standard, m[1], text)
					}
				}
			}
			if strings.Contains(it.Prompt, "milliliter") {
				t.Fatalf("milliliters are not a grade 3 unit: %s", it.Prompt)
			}
		}
	}
}
