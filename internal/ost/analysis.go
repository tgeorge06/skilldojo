package ost

import (
	"sort"
	"time"
)

// Focus is one area a child should work on, with where in the app to do it.
type Focus struct {
	Category  string   `json:"category"`
	Percent   int      `json:"percent"` // recent weighted percent
	Attempts  int      `json:"attempts"`
	Standards []Tally  `json:"standards"` // weakest standards inside the category
	Practice  Practice `json:"practice"`
}

// Practice points at a mode in the app that drills a category.
type Practice struct {
	Label string `json:"label"`
	Path  string `json:"path"` // empty when the app has no drill yet
	Note  string `json:"note,omitempty"`
}

// Trend is one category's percent across finished attempts, oldest first.
type Trend struct {
	Category string `json:"category"`
	Points   []int  `json:"points"`
}

// Analysis is what the parent report derives from a child's history.
type Analysis struct {
	Finished   int
	Latest     *Summary
	Best       int
	Average    int
	Trend      []int // overall percent per finished attempt, oldest first
	Categories []Trend
	Focus      []Focus // weakest first
	Strong     []string
}

// AnalysisGrade picks which test grade to analyse: the grade of the most
// recent finished attempt, so a child who moved up is judged on the new
// test. Falls back to the child's nearest grade.
func AnalysisGrade(history []Summary, fallback int) int {
	grade, latest := fallback, time.Time{}
	for _, h := range history {
		if h.Finished && h.FinishedAt.After(latest) {
			grade, latest = h.Grade, h.FinishedAt
		}
	}
	return grade
}

// OfGrade keeps only one grade's attempts; results from different grades
// are different tests and must not be averaged together.
func OfGrade(history []Summary, grade int) []Summary {
	var out []Summary
	for _, h := range history {
		if h.Grade == grade {
			out = append(out, h)
		}
	}
	return out
}

// NeedsWorkBelow is the recent-percent line under which a category is
// flagged. Proficient on the real test sits near the middle of the scale.
const NeedsWorkBelow = 70

// Analyze folds a history into trends and focus areas. Recent attempts
// count more (weights 3,2,1,1,...) so a child who has improved is not
// nagged about an old test.
func Analyze(history []Summary, grade int) Analysis {
	var an Analysis
	var finished []Summary
	for _, h := range history {
		if h.Finished {
			finished = append(finished, h)
		}
	}
	an.Finished = len(finished)
	if len(finished) == 0 {
		return an
	}
	an.Latest = &finished[0]
	sum := 0
	for i := len(finished) - 1; i >= 0; i-- { // oldest first
		h := finished[i]
		an.Trend = append(an.Trend, h.Percent)
		sum += h.Percent
		an.Best = max(an.Best, h.Percent)
	}
	an.Average = sum / len(finished)

	type acc struct {
		wRight, wTotal float64
		attempts       int
		stds           map[string]*Tally
		points         []int
	}
	cats := map[string]*acc{}
	order := Categories(grade)
	for _, c := range order {
		cats[c] = &acc{stds: map[string]*Tally{}}
	}
	for i, h := range finished { // newest first; weight 3,2,1,1,...
		w := float64(max(3-i, 1))
		for _, t := range h.Categories {
			a := cats[t.Name]
			if a == nil {
				a = &acc{stds: map[string]*Tally{}}
				cats[t.Name] = a
				order = append(order, t.Name)
			}
			a.wRight += w * float64(t.Right)
			a.wTotal += w * float64(t.Total)
			a.attempts++
		}
		for _, t := range h.Standards {
			// A standard belongs to one reporting category; look it up in
			// the template catalogue.
			c := standardCategory(grade, t.Name)
			a := cats[c]
			if a == nil {
				continue
			}
			st := a.stds[t.Name]
			if st == nil {
				st = &Tally{Name: t.Name}
				a.stds[t.Name] = st
			}
			st.Right += t.Right
			st.Total += t.Total
		}
	}
	for i := len(finished) - 1; i >= 0; i-- {
		byName := map[string]int{}
		for _, t := range finished[i].Categories {
			byName[t.Name] = t.Percent
		}
		for _, c := range order {
			if p, ok := byName[c]; ok {
				cats[c].points = append(cats[c].points, p)
			}
		}
	}
	for _, c := range order {
		a := cats[c]
		if a.attempts == 0 {
			continue
		}
		an.Categories = append(an.Categories, Trend{Category: c, Points: a.points})
		p := int(a.wRight / a.wTotal * 100)
		if p >= NeedsWorkBelow {
			an.Strong = append(an.Strong, c)
			continue
		}
		f := Focus{Category: c, Percent: p, Attempts: a.attempts, Practice: practiceFor(c)}
		for _, st := range a.stds {
			st.Percent = pct(st.Right, st.Total)
			if st.Percent < NeedsWorkBelow {
				f.Standards = append(f.Standards, *st)
			}
		}
		sort.Slice(f.Standards, func(i, j int) bool {
			if f.Standards[i].Percent != f.Standards[j].Percent {
				return f.Standards[i].Percent < f.Standards[j].Percent
			}
			return f.Standards[i].Name < f.Standards[j].Name
		})
		if len(f.Standards) > 4 {
			f.Standards = f.Standards[:4]
		}
		an.Focus = append(an.Focus, f)
	}
	sort.SliceStable(an.Focus, func(i, j int) bool { return an.Focus[i].Percent < an.Focus[j].Percent })
	return an
}

// standardCategory finds which reporting category a standard was
// registered under for a grade.
func standardCategory(grade int, standard string) string {
	for _, t := range templates[grade] {
		if t.Standard == standard {
			return t.Category
		}
	}
	return ""
}

// practiceFor maps a reporting category onto the app's drills. Paths use
// the practice page's hash so the dojo opens ready to go.
func practiceFor(category string) Practice {
	switch category {
	case "Multiplication and Division":
		return Practice{Label: "Times tables and ×/÷ sheets", Path: "/#math/tables"}
	case "Number and Operations":
		return Practice{Label: "Add & subtract sheets", Path: "/#math/addsub"}
	case "Fractions":
		return Practice{Label: "Fraction sheets", Path: "/#math/frac"}
	case "Decimals":
		return Practice{Label: "Practice tests", Note: "The math dojo has no decimal drill yet; review missed items below and take another test."}
	case "Geometry":
		return Practice{Label: "Practice tests", Note: "Geometry is not a dojo drill yet; review the explanations for missed items together."}
	}
	return Practice{Label: "Practice tests"}
}

// StandardLabel gives a parent-readable name for a standard code.
func StandardLabel(code string) string {
	if l, ok := standardLabels[code]; ok {
		return l
	}
	return code
}

var standardLabels = map[string]string{
	"3.OA.1": "Meaning of multiplication", "3.OA.2": "Meaning of division", "3.OA.3": "Multiply and divide word problems",
	"3.OA.4": "Unknown numbers in × and ÷", "3.OA.5": "Properties of multiplication", "3.OA.6": "Division as unknown factor",
	"3.OA.7": "Fluent × and ÷ within 100", "3.OA.8": "Two-step word problems", "3.OA.9": "Arithmetic patterns",
	"3.NBT.1": "Rounding to 10 and 100", "3.NBT.2": "Add and subtract within 1,000", "3.NBT.3": "Multiply by multiples of 10",
	"3.NF.1": "Understanding fractions", "3.NF.2": "Fractions on a number line", "3.NF.3": "Equivalent fractions and comparing",
	"3.MD.1": "Time to the minute", "3.MD.2": "Mass and volume", "3.MD.5": "Area concepts", "3.MD.7": "Area of rectangles", "3.MD.8": "Perimeter",
	"3.G.1": "Shapes and their attributes", "3.G.2": "Partitioning shapes into equal areas",
	"4.OA.1": "Multiplicative comparison", "4.OA.2": "Comparison word problems", "4.OA.3": "Multi-step word problems and remainders",
	"4.OA.4": "Factors, multiples, primes", "4.OA.5": "Number patterns",
	"4.NBT.1": "Place value ten times", "4.NBT.2": "Compare multi-digit numbers", "4.NBT.3": "Rounding multi-digit numbers",
	"4.NBT.4": "Add and subtract multi-digit numbers", "4.NBT.5": "Multi-digit multiplication", "4.NBT.6": "Division with one-digit divisors",
	"4.NF.1": "Equivalent fractions", "4.NF.2": "Comparing fractions", "4.NF.3": "Add and subtract fractions", "4.NF.4": "Multiply fractions by whole numbers",
	"4.NF.5": "Tenths and hundredths", "4.NF.6": "Fractions as decimals", "4.NF.7": "Comparing decimals",
	"4.MD.1": "Unit conversions", "4.MD.2": "Measurement word problems", "4.MD.3": "Area and perimeter formulas", "4.MD.4": "Line plots with fractions",
	"4.MD.5": "Angle concepts", "4.MD.6": "Measuring angles", "4.MD.7": "Adding angles",
	"4.G.1": "Lines, rays, angles", "4.G.2": "Classifying shapes", "4.G.3": "Lines of symmetry",
	"5.OA.1": "Order of operations", "5.OA.2": "Writing expressions", "5.OA.3": "Comparing patterns",
	"5.NBT.1": "Decimal place value", "5.NBT.2": "Powers of ten", "5.NBT.3": "Reading and comparing decimals", "5.NBT.4": "Rounding decimals",
	"5.NBT.5": "Multi-digit multiplication", "5.NBT.6": "Division with two-digit divisors", "5.NBT.7": "Decimal operations",
	"5.NF.1": "Add and subtract unlike fractions", "5.NF.2": "Fraction word problems", "5.NF.3": "Fractions as division",
	"5.NF.4": "Multiplying fractions", "5.NF.5": "Multiplication as scaling", "5.NF.6": "Fraction multiplication word problems", "5.NF.7": "Dividing with unit fractions",
	"5.MD.1": "Metric conversions", "5.MD.2": "Line plots with fractions", "5.MD.3": "Volume concepts", "5.MD.4": "Measuring volume", "5.MD.5": "Volume of prisms",
	"5.G.1": "Coordinate plane", "5.G.2": "Graphing points", "5.G.3": "Properties of 2-D shapes", "5.G.4": "Classifying 2-D shapes",
}
