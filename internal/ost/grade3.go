package ost

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Grade 3 templates. Categories: Multiplication and Division (3.OA),
// Number and Operations (3.NBT, 3.MD time and measurement), Fractions
// (3.NF), Geometry (3.G, 3.MD area and perimeter).
func init() {
	const md, no, fr, ge = "Multiplication and Division", "Number and Operations", "Fractions", "Geometry"
	register(3,
		Template{md, "3.OA.1", 1, func(r *rand.Rand) Item {
			g, n := between(r, 2, 9), between(r, 2, 9)
			thing := pick(r, things)
			return choices(r, fmt.Sprintf("There are %d %s. Each one has %d %s inside. Which math sentence (expression) shows how many %s there are in all?", g, pick(r, containers), n, thing, thing),
				fmt.Sprintf("%d × %d", g, n), []string{fmt.Sprintf("%d + %d", g, n), fmt.Sprintf("%d − %d", g+n, n), fmt.Sprintf("%d ÷ %d", g*n, g+1)},
				fmt.Sprintf("%d groups of %d is %d × %d.", g, n, g, n))
		}},
		Template{md, "3.OA.2", 2, func(r *rand.Rand) Item {
			g, n := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("%d friends share %d %s. Each friend gets the same amount. How many does each friend get?", g, g*n, pick(r, things)),
				itoa(n), fmt.Sprintf("%d ÷ %d = %d.", g*n, g, n))
		}},
		Template{md, "3.OA.3", 2, func(r *rand.Rand) Item {
			rows, per := between(r, 3, 9), between(r, 3, 9)
			return figure(numeric(fmt.Sprintf("A garden has %d rows of plants. Each row has %d plants. How many plants are there in all?", rows, per),
				itoa(rows*per), fmt.Sprintf("%d × %d = %d.", rows, per, rows*per)), Figure{Kind: "grid", A: per, B: rows})
		}},
		Template{md, "3.OA.4", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("What number goes in the box to make this true?  %d × ▢ = %d", a, a*b), itoa(b), fmt.Sprintf("%d × %d = %d.", a, b, a*b))
		}},
		Template{md, "3.OA.5", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			for b == a { // a × a reads the same either way, and 2 + 2 = 2 × 2
				b = between(r, 2, 9)
			}
			return choices(r, fmt.Sprintf("Which one is the same as %d × %d?", a, b),
				fmt.Sprintf("%d × %d", b, a), []string{fmt.Sprintf("%d + %d", a, b), fmt.Sprintf("%d × %d", a, b+1), fmt.Sprintf("%d ÷ %d", a*b, b+1)},
				"Changing the order of the factors does not change the product.")
		}},
		Template{md, "3.OA.6", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			for b == a { // a × a would duplicate the correct fact
				b = between(r, 2, 9)
			}
			return choices(r, fmt.Sprintf("Which times fact helps you solve %d ÷ %d?", a*b, a),
				fmt.Sprintf("%d × %d = %d", a, b, a*b), []string{fmt.Sprintf("%d × %d = %d", a, a, a*a), fmt.Sprintf("%d + %d = %d", a, b, a+b), fmt.Sprintf("%d × %d = %d", a*b, a, a*b*a)},
				fmt.Sprintf("%d ÷ %d asks: %d times what makes %d?", a*b, a, a, a*b))
		}},
		Template{md, "3.OA.7", 1, func(r *rand.Rand) Item {
			a, b := between(r, 2, 10), between(r, 2, 10)
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), fmt.Sprintf("%d × %d = %d.", a, b, a*b))
		}},
		Template{md, "3.OA.7", 1, func(r *rand.Rand) Item {
			a, b := between(r, 2, 10), between(r, 2, 10)
			return numeric(fmt.Sprintf("%d ÷ %d = ?", a*b, b), itoa(a), fmt.Sprintf("%d ÷ %d = %d because %d × %d = %d.", a*b, b, a, a, b, a*b))
		}},
		Template{md, "3.OA.8", 3, func(r *rand.Rand) Item {
			packs, per, gave := between(r, 3, 6), between(r, 4, 8), between(r, 2, 9)
			total := packs*per - gave
			return numeric(fmt.Sprintf("%s buys %d packs of %s. Each pack has %d. Then %s gives %d away. How many are left?", pick(r, names), packs, pick(r, things), per, "the child", gave),
				itoa(total), fmt.Sprintf("%d × %d = %d, then %d − %d = %d.", packs, per, packs*per, packs*per, gave, total))
		}},
		Template{md, "3.OA.9", 2, func(r *rand.Rand) Item {
			step := between(r, 3, 9)
			start := step * between(r, 1, 3)
			seq := fmt.Sprintf("%d, %d, %d, %d, ▢", start, start+step, start+2*step, start+3*step)
			return numeric(fmt.Sprintf("What number comes next in this pattern?  %s", seq), itoa(start+4*step), fmt.Sprintf("Each number is %d more than the one before.", step))
		}},
		Template{no, "3.NBT.1", 1, func(r *rand.Rand) Item {
			n := between(r, 105, 995)
			rounded := ((n + 5) / 10) * 10
			return numeric(fmt.Sprintf("Round %d to the nearest ten. (Which ten is it closest to?)", n), itoa(rounded), fmt.Sprintf("%d is closest to %d.", n, rounded))
		}},
		Template{no, "3.NBT.1", 1, func(r *rand.Rand) Item {
			n := between(r, 150, 949)
			rounded := ((n + 50) / 100) * 100
			return choices(r, fmt.Sprintf("Round %d to the nearest hundred. Which hundred is it closest to?", n),
				itoa(rounded), []string{itoa(((n + 5) / 10) * 10), itoa(rounded + 100), itoa(rounded - 100), itoa(rounded + 50)},
				fmt.Sprintf("Look at the tens digit of %d to decide whether to round up or down.", n))
		}},
		Template{no, "3.NBT.2", 1, func(r *rand.Rand) Item {
			a, b := between(r, 120, 480), between(r, 120, 480)
			return numeric(fmt.Sprintf("%d + %d = ?", a, b), itoa(a+b), fmt.Sprintf("%d + %d = %d.", a, b, a+b))
		}},
		Template{no, "3.NBT.2", 2, func(r *rand.Rand) Item {
			a, b := between(r, 500, 980), between(r, 120, 480)
			return numeric(fmt.Sprintf("A library had %d books. People borrowed %d of them. How many books are still there?", a, b), itoa(a-b), fmt.Sprintf("%d − %d = %d.", a, b, a-b))
		}},
		Template{md, "3.NBT.3", 1, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)*10
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), fmt.Sprintf("%d × %d tens = %d tens = %d.", a, b/10, a*b/10, a*b))
		}},
		Template{no, "3.NBT.2", 2, func(r *rand.Rand) Item {
			a, b, c := between(r, 100, 300), between(r, 100, 300), between(r, 100, 300)
			return numeric(fmt.Sprintf("Three classes collected cans for a food drive: %d cans, %d cans, and %d cans. How many cans is that in all?", a, b, c), itoa(a+b+c), "Add the three amounts.")
		}},
		Template{no, "3.MD.1", 2, func(r *rand.Rand) Item {
			h, m, add := between(r, 1, 11), between(r, 0, 5)*10, between(r, 15, 45)
			end := h*60 + m + add
			eh, em := (end/60-1)%12+1, end%60
			return choices(r, fmt.Sprintf("Soccer practice starts at %d:%02d. It lasts %d minutes. What time is it over?", h, m, add),
				fmt.Sprintf("%d:%02d", eh, em), []string{fmt.Sprintf("%d:%02d", eh, (em+10)%60), fmt.Sprintf("%d:%02d", (eh%12)+1, em), fmt.Sprintf("%d:%02d", h, (m+add+5)%60)},
				fmt.Sprintf("Add %d minutes to %d:%02d.", add, h, m))
		}},
		Template{no, "3.MD.2", 2, func(r *rand.Rand) Item {
			each, n := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("One jug holds %d liters of juice. How many liters of juice are in %d jugs?", each, n), itoa(each*n), fmt.Sprintf("%d × %d = %d liters.", each, n, each*n))
		}},
		Template{no, "3.MD.2", 2, func(r *rand.Rand) Item {
			a, b := between(r, 120, 480), between(r, 120, 480)
			return numeric(fmt.Sprintf("A bag of apples has a mass of %d grams. A bag of pears has a mass of %d grams. How many grams do the two bags have together?", a, b), itoa(a+b), fmt.Sprintf("%d + %d = %d grams.", a, b, a+b))
		}},
		Template{no, "3.MD.2", 2, func(r *rand.Rand) Item {
			total, n := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("%d books together have a mass of %d kilograms. Each book has the same mass. What is the mass of one book, in kilograms?", n, total*n), itoa(total), fmt.Sprintf("%d ÷ %d = %d kilograms.", total*n, n, total))
		}},
		Template{no, "3.MD.3", 2, func(r *rand.Rand) Item {
			scale := pick(r, []int{2, 5, 10})
			names := []string{"Dogs", "Cats", "Fish", "Birds"}
			vals := []int{between(r, 1, 6) * scale, between(r, 1, 6) * scale, between(r, 1, 6) * scale, between(r, 1, 6) * scale}
			i, j := 0, 1
			for vals[i] == vals[j] {
				vals[j] = between(r, 1, 6) * scale
			}
			hi, lo := i, j
			if vals[lo] > vals[hi] {
				hi, lo = lo, hi
			}
			return figure(numeric(fmt.Sprintf("The bar graph shows the pets in Ms. Lee's class. How many more %s than %s are there?", strings.ToLower(names[hi]), strings.ToLower(names[lo])), itoa(vals[hi]-vals[lo]),
				fmt.Sprintf("%s: %d, %s: %d. %d − %d = %d.", names[hi], vals[hi], names[lo], vals[lo], vals[hi], vals[lo], vals[hi]-vals[lo])),
				Figure{Kind: "bars", A: scale, Names: names, Values: vals})
		}},
		Template{no, "3.MD.3", 3, func(r *rand.Rand) Item {
			scale := pick(r, []int{2, 5})
			names := []string{"Red", "Blue", "Green"}
			vals := []int{between(r, 1, 6) * scale, between(r, 1, 6) * scale, between(r, 1, 6) * scale}
			return figure(numeric("The bar graph shows the favorite colors of the students in a class. How many students are in the class?", itoa(vals[0]+vals[1]+vals[2]),
				fmt.Sprintf("%d + %d + %d = %d students.", vals[0], vals[1], vals[2], vals[0]+vals[1]+vals[2])),
				Figure{Kind: "bars", A: scale, Names: names, Values: vals})
		}},
		Template{fr, "3.NF.1", 1, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			os := otherDenominators(r, d)
			return figure(choices(r, fmt.Sprintf("A pizza is cut into %d equal slices. %s eats %d slices. What fraction of the pizza is that?", d, pick(r, names), n),
				frac(n, d), []string{frac(d-n, d), frac(n, os[0]), frac(n, os[1]), frac(n, os[2]), frac(n, os[3]), frac(n+1, d)}, fmt.Sprintf("%d of %d equal parts is %s.", n, d, frac(n, d))), Figure{Kind: "parts", A: d, B: n})
		}},
		Template{fr, "3.NF.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			os := otherDenominators(r, d)
			return figure(choices(r, fmt.Sprintf("This number line goes from 0 to 1. It is cut into %d equal parts. What fraction is the dot on?", d),
				frac(n, d), []string{frac(n, os[0]), frac(d-n, d), frac(n+1, d), frac(n, os[1]), frac(n, os[2]), frac(n, os[3])}, "Count the equal parts from 0 to the dot."), Figure{Kind: "numberline", A: d, B: n})
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			// Equivalent pairs whose denominators stay within 2, 3, 4, 6, 8.
			pairs := [][4]int{{1, 2, 2, 4}, {1, 2, 3, 6}, {1, 2, 4, 8}, {1, 3, 2, 6}, {2, 3, 4, 6}, {1, 4, 2, 8}, {3, 4, 6, 8}, {2, 4, 4, 8}}
			pr := pairs[r.IntN(len(pairs))]
			n, d, n2, d2 := pr[0], pr[1], pr[2], pr[3]
			return figure(choices(r, fmt.Sprintf("Which fraction is the same amount as %s? (The same amount is called equivalent.)", frac(n, d)),
				frac(n2, d2), []string{frac(n2+1, d2), frac(n, d2), frac(n2, d), frac(d2-n2, d2), frac(n+1, d2), frac(n2-1, d2)}, "Multiply the top and bottom by the same number."), Figure{Kind: "parts", A: d, B: n})
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{3, 4, 6, 8})
			a, b := between(r, 1, d-1), between(r, 1, d-1)
			for b == a {
				b = between(r, 1, d-1)
			}
			big, small := frac(max(a, b), d), frac(min(a, b), d)
			return choices(r, "Which one is true? (> means bigger, < means smaller)",
				fmt.Sprintf("%s > %s", big, small), []string{fmt.Sprintf("%s < %s", big, small), fmt.Sprintf("%s = %s", big, small), fmt.Sprintf("%s > %s", small, big)},
				"When the bottom numbers match, the fraction with the bigger top number is bigger.")
		}},
		Template{fr, "3.NF.3", 1, func(r *rand.Rand) Item {
			d := pick(r, []int{3, 4, 6, 8})
			os := otherDenominators(r, d)
			return choices(r, "Which fraction is the same as 1 whole?", frac(d, d), []string{frac(1, d), frac(d-1, d), frac(d, os[0]), frac(1, os[1]), frac(d, os[2]), frac(1, os[3])}, "When the top and bottom numbers match, you have all the parts: 1 whole.")
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			sets := []struct {
				target string
				opts   []string
				right  []int
			}{
				{"1/2", []string{"2/4", "3/6", "1/3", "4/8", "2/3"}, []int{0, 1, 3}},
				{"1/3", []string{"2/6", "1/4", "3/6", "2/8", "3/4"}, nil},
				{"1/4", []string{"2/8", "2/4", "1/2", "3/8", "1/3"}, nil},
				{"2/3", []string{"4/6", "3/4", "2/4", "5/8", "1/3"}, nil},
				{"3/4", []string{"6/8", "3/8", "4/6", "2/3", "1/2"}, nil},
			}
			st := sets[r.IntN(len(sets))]
			right := st.right
			if right == nil {
				for i, o := range st.opts {
					if sameNumber(o, st.target) {
						right = append(right, i)
					}
				}
			}
			return multi(fmt.Sprintf("Which fractions are the same amount as %s?", st.target), st.opts, right, fmt.Sprintf("Each right answer is another name for %s.", st.target))
		}},
		Template{fr, "3.MD.4", 2, func(r *rand.Rand) Item {
			// Pencil lengths in inches to the nearest quarter, as X marks.
			vals := make([]int, 9)
			total := 0
			for i := 2; i <= 8; i++ {
				vals[i] = between(r, 0, 3)
				total += vals[i]
			}
			if total == 0 {
				vals[4] = 2
				total = 2
			}
			cut := pick(r, []int{4, 6}) // 1 inch or 1 1/2 inches
			longer := 0
			for i := cut + 1; i <= 8; i++ {
				longer += vals[i]
			}
			cutLabel := map[int]string{4: "1 inch", 6: "1 1/2 inches"}[cut]
			return figure(numeric(fmt.Sprintf("The line plot shows how long some pencils are. How many pencils are longer than %s?", cutLabel), itoa(longer),
				fmt.Sprintf("Count the X marks to the right of %s: %d.", cutLabel, longer)), Figure{Kind: "lineplot", A: 8, Values: vals})
		}},
		Template{ge, "3.MD.7", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 9), between(r, 2, 8)
			if w > l {
				l, w = w, l // "long" is never the shorter side
			}
			return figure(numeric(fmt.Sprintf("This rectangle is %d squares long and %d squares wide. How many squares fit inside it? (That is its area.)", l, w), itoa(l*w), fmt.Sprintf("Area = %d × %d = %d squares.", l, w, l*w)), Figure{Kind: "grid", A: l, B: w})
		}},
		Template{ge, "3.MD.8", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 12), between(r, 2, 9)
			if w > l {
				l, w = w, l // "long" is never the shorter side
			}
			return figure(numeric(fmt.Sprintf("A garden is shaped like a rectangle. It is %d feet long and %d feet wide. How many feet of fence go all the way around it? (That distance is the perimeter.)", l, w), itoa(2*(l+w)), fmt.Sprintf("Perimeter = %d + %d + %d + %d = %d.", l, w, l, w, 2*(l+w))), Figure{Kind: "rect", A: l, B: w, LabelA: fmt.Sprintf("%d ft", l), LabelB: fmt.Sprintf("%d ft", w)})
		}},
		Template{ge, "3.MD.5", 1, func(r *rand.Rand) Item {
			n := between(r, 6, 24)
			return numeric(fmt.Sprintf("A shape is covered with %d little squares. There are no gaps and none overlap. What is the area of the shape, in squares?", n), itoa(n), "Area is how many squares cover the shape.")
		}},
		Template{ge, "3.MD.6", 1, func(r *rand.Rand) Item {
			a, b := between(r, 3, 9), between(r, 2, 6)
			if b > a {
				a, b = b, a
			}
			unit := pick(r, []string{"centimeter", "meter", "inch"})
			plural := unit + "s"
			if unit == "inch" {
				plural = "inches"
			}
			return figure(numeric(fmt.Sprintf("Each small square is 1 square %s. What is the area of the shape, in square %s? Count the squares.", unit, plural), itoa(a*b), fmt.Sprintf("%d rows of %d squares is %d square %s.", b, a, a*b, plural)), Figure{Kind: "grid", A: a, B: b})
		}},
		Template{ge, "3.G.1", 1, func(r *rand.Rand) Item {
			quads := []string{"square", "rectangle", "rhombus"}
			others := []string{"triangle", "hexagon", "pentagon", "octagon", "circle"}
			r.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
			opts := append(append([]string{}, quads...), others[:2]...)
			r.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
			var right []int
			for i, o := range opts {
				if o == "square" || o == "rectangle" || o == "rhombus" {
					right = append(right, i)
				}
			}
			return multi("Which shapes have exactly 4 sides? (A 4-sided shape is a quadrilateral.)", opts, right, "A quadrilateral has exactly four sides.")
		}},
		Template{ge, "3.G.1", 2, func(r *rand.Rand) Item {
			kind := r.IntN(3)
			switch kind {
			case 0:
				return choices(r, "A shape has 4 sides that are all the same length, and 4 square corners. What is it?", "square", []string{"rectangle that is not a square", "triangle", "pentagon"}, "Four equal sides and four square corners make a square.")
			case 1:
				return choices(r, "A shape has 4 sides and 4 square corners. Two sides are long and two sides are short. What is it?", "rectangle", []string{"square", "rhombus", "triangle"}, "Four square corners with two long and two short sides make a rectangle.")
			}
			return choices(r, "A shape has 4 sides that are all the same length, but its corners are not square. What is it?", "rhombus", []string{"square", "rectangle", "hexagon"}, "Four equal sides without square corners make a rhombus.")
		}},
		Template{ge, "3.G.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			os := otherDenominators(r, d)
			return figure(choices(r, fmt.Sprintf("This rectangle is cut into %d equal parts. What fraction of the rectangle is one part?", d), frac(1, d), []string{frac(d, d), frac(1, os[0]), frac(1, os[1]), frac(1, os[2]), frac(1, os[3]), frac(2, d), frac(d-1, d)}, fmt.Sprintf("Each of %d equal parts is 1/%d of the whole.", d, d)), Figure{Kind: "parts", A: d, B: 1})
		}},
		Template{ge, "3.MD.8", 3, func(r *rand.Rand) Item {
			l := between(r, 6, 9)
			w := between(r, 2, 5) // always shorter than l, so "long" and "short" are true
			p := 2 * (l + w)
			return figure(numeric(fmt.Sprintf("A fence goes all the way around a rectangle-shaped yard. The whole fence is %d feet long. (That is the perimeter.) The long side is %d feet. How long is the short side, in feet?", p, l), itoa(w), fmt.Sprintf("Two long sides use %d feet. %d − %d = %d feet for the two short sides, so each short side is %d.", 2*l, p, 2*l, p-2*l, w)), Figure{Kind: "rect", A: l, B: w, LabelA: fmt.Sprintf("%d ft", l), LabelB: "?"})
		}},
	)
}

// otherDenominators lists the allowed grade 3 denominators other than d,
// in random order.
func otherDenominators(r *rand.Rand, d int) []int {
	var out []int
	for _, o := range grade3Denominators {
		if o != d {
			out = append(out, o)
		}
	}
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// otherDenominator picks an allowed grade 3 denominator different from d.
func otherDenominator(r *rand.Rand, d int) int {
	for {
		o := grade3Denominators[r.IntN(len(grade3Denominators))]
		if o != d {
			return o
		}
	}
}
