package ost

import (
	"fmt"
	"math/rand/v2"
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
			return choices(r, fmt.Sprintf("There are %d %s with %d %s in each. Which expression shows the total number of %s?", g, pick(r, containers), n, thing, thing),
				fmt.Sprintf("%d × %d", g, n), []string{fmt.Sprintf("%d + %d", g, n), fmt.Sprintf("%d − %d", g+n, n), fmt.Sprintf("%d ÷ %d", g*n, g+1)},
				fmt.Sprintf("%d groups of %d is %d × %d.", g, n, g, n))
		}},
		Template{md, "3.OA.2", 2, func(r *rand.Rand) Item {
			g, n := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("%s shares %d %s equally among %d friends. How many does each friend get?", pick(r, names), g*n, pick(r, things), g),
				itoa(n), fmt.Sprintf("%d ÷ %d = %d.", g*n, g, n))
		}},
		Template{md, "3.OA.3", 2, func(r *rand.Rand) Item {
			rows, per := between(r, 3, 9), between(r, 3, 9)
			return numeric(fmt.Sprintf("A garden has %d rows of plants with %d plants in each row. How many plants are in the garden?", rows, per),
				itoa(rows*per), fmt.Sprintf("%d × %d = %d.", rows, per, rows*per))
		}},
		Template{md, "3.OA.4", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("What number makes this equation true?  %d × ▢ = %d", a, a*b), itoa(b), fmt.Sprintf("%d × %d = %d.", a, b, a*b))
		}},
		Template{md, "3.OA.5", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			return choices(r, fmt.Sprintf("Which expression is equal to %d × %d?", a, b),
				fmt.Sprintf("%d × %d", b, a), []string{fmt.Sprintf("%d + %d", a, b), fmt.Sprintf("%d × %d", a, b+1), fmt.Sprintf("%d ÷ %d", a*b, b+1)},
				"Changing the order of the factors does not change the product.")
		}},
		Template{md, "3.OA.6", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)
			for b == a { // a × a would duplicate the correct fact
				b = between(r, 2, 9)
			}
			return choices(r, fmt.Sprintf("Which multiplication fact can be used to solve %d ÷ %d?", a*b, a),
				fmt.Sprintf("%d × %d = %d", a, b, a*b), []string{fmt.Sprintf("%d × %d = %d", a, a, a*a), fmt.Sprintf("%d + %d = %d", a, b, a+b), fmt.Sprintf("%d × %d = %d", a*b, a, a*b*a)},
				"Division is the unknown-factor problem: a × ? = the dividend.")
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
			return numeric(fmt.Sprintf("%s buys %d packs of %s with %d in each pack, then gives %d away. How many are left?", pick(r, names), packs, pick(r, things), per, gave),
				itoa(total), fmt.Sprintf("%d × %d = %d, then %d − %d = %d.", packs, per, packs*per, packs*per, gave, total))
		}},
		Template{md, "3.OA.9", 2, func(r *rand.Rand) Item {
			step := between(r, 3, 9)
			start := step * between(r, 1, 3)
			seq := fmt.Sprintf("%d, %d, %d, %d, ▢", start, start+step, start+2*step, start+3*step)
			return numeric(fmt.Sprintf("What is the next number in the pattern?  %s", seq), itoa(start+4*step), fmt.Sprintf("Each number is %d more than the one before.", step))
		}},
		Template{no, "3.NBT.1", 1, func(r *rand.Rand) Item {
			n := between(r, 105, 995)
			rounded := ((n + 5) / 10) * 10
			return numeric(fmt.Sprintf("Round %d to the nearest ten.", n), itoa(rounded), fmt.Sprintf("%d is closest to %d.", n, rounded))
		}},
		Template{no, "3.NBT.1", 1, func(r *rand.Rand) Item {
			n := between(r, 150, 949)
			rounded := ((n + 50) / 100) * 100
			return choices(r, fmt.Sprintf("Which number is %d rounded to the nearest hundred?", n),
				itoa(rounded), []string{itoa(((n + 5) / 10) * 10), itoa(rounded + 100), itoa(rounded - 100), itoa(rounded + 50)},
				fmt.Sprintf("Look at the tens digit of %d to decide whether to round up or down.", n))
		}},
		Template{no, "3.NBT.2", 1, func(r *rand.Rand) Item {
			a, b := between(r, 120, 480), between(r, 120, 480)
			return numeric(fmt.Sprintf("%d + %d = ?", a, b), itoa(a+b), fmt.Sprintf("%d + %d = %d.", a, b, a+b))
		}},
		Template{no, "3.NBT.2", 2, func(r *rand.Rand) Item {
			a, b := between(r, 500, 980), between(r, 120, 480)
			return numeric(fmt.Sprintf("A library had %d books. It lent out %d. How many books are left?", a, b), itoa(a-b), fmt.Sprintf("%d − %d = %d.", a, b, a-b))
		}},
		Template{no, "3.NBT.3", 1, func(r *rand.Rand) Item {
			a, b := between(r, 2, 9), between(r, 2, 9)*10
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), fmt.Sprintf("%d × %d tens = %d tens = %d.", a, b/10, a*b/10, a*b))
		}},
		Template{no, "3.NBT.2", 2, func(r *rand.Rand) Item {
			a, b, c := between(r, 100, 300), between(r, 100, 300), between(r, 100, 300)
			return numeric(fmt.Sprintf("Three classes collected %d, %d, and %d cans for a food drive. How many cans did they collect in all?", a, b, c), itoa(a+b+c), "Add the three amounts.")
		}},
		Template{no, "3.MD.1", 2, func(r *rand.Rand) Item {
			h, m, add := between(r, 1, 11), between(r, 0, 5)*10, between(r, 15, 45)
			end := h*60 + m + add
			eh, em := (end/60-1)%12+1, end%60
			return choices(r, fmt.Sprintf("Practice starts at %d:%02d and lasts %d minutes. What time does it end?", h, m, add),
				fmt.Sprintf("%d:%02d", eh, em), []string{fmt.Sprintf("%d:%02d", eh, (em+10)%60), fmt.Sprintf("%d:%02d", (eh%12)+1, em), fmt.Sprintf("%d:%02d", h, (m+add+5)%60)},
				fmt.Sprintf("Add %d minutes to %d:%02d.", add, h, m))
		}},
		Template{no, "3.MD.2", 2, func(r *rand.Rand) Item {
			each, n := between(r, 200, 450), between(r, 2, 4)
			return numeric(fmt.Sprintf("One bottle holds %d milliliters of juice. How many milliliters do %d bottles hold?", each, n), itoa(each*n), fmt.Sprintf("%d × %d = %d.", each, n, each*n))
		}},
		Template{fr, "3.NF.1", 1, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			return choices(r, fmt.Sprintf("A pizza is cut into %d equal slices. %s eats %d of them. What fraction of the pizza is that?", d, pick(r, names), n),
				frac(n, d), []string{frac(d, n), frac(n, d+1), frac(n, d-1)}, fmt.Sprintf("%d of %d equal parts is %s.", n, d, frac(n, d)))
		}},
		Template{fr, "3.NF.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			return choices(r, fmt.Sprintf("A number line from 0 to 1 is divided into %d equal parts. Which fraction names the point %d parts from 0?", d, n),
				frac(n, d), []string{frac(n, d*2), frac(d, n), frac(n+1, d)}, "Count the equal parts from 0.")
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4})
			n := between(r, 1, d-1)
			k := pick(r, []int{2, 3})
			return choices(r, fmt.Sprintf("Which fraction is equivalent to %s?", frac(n, d)),
				frac(n*k, d*k), []string{frac(n+1, d+1), frac(n*k, d), frac(n, d*k)}, "Multiply the numerator and denominator by the same number.")
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{3, 4, 6, 8})
			a, b := between(r, 1, d-1), between(r, 1, d-1)
			for b == a {
				b = between(r, 1, d-1)
			}
			big, small := frac(max(a, b), d), frac(min(a, b), d)
			return choices(r, "Which comparison is true?",
				fmt.Sprintf("%s > %s", big, small), []string{fmt.Sprintf("%s < %s", big, small), fmt.Sprintf("%s = %s", big, small), fmt.Sprintf("%s > %s", small, big)},
				"With the same denominator, the fraction with more parts is greater.")
		}},
		Template{fr, "3.NF.3", 1, func(r *rand.Rand) Item {
			d := pick(r, []int{3, 4, 5, 6, 8})
			return choices(r, "Which fraction is equal to 1 whole?", frac(d, d), []string{frac(1, d), frac(d, 1), frac(d-1, d)}, "A fraction with the same numerator and denominator equals 1.")
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			return multi("Which fractions are equal to 1/2?", []string{"2/4", "3/6", "1/3", "4/8", "2/3"}, []int{0, 1, 3}, "2/4, 3/6, and 4/8 all simplify to 1/2.")
		}},
		Template{ge, "3.MD.7", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 9), between(r, 2, 8)
			return numeric(fmt.Sprintf("A rectangle is %d units long and %d units wide. What is its area in square units?", l, w), itoa(l*w), fmt.Sprintf("Area = length × width = %d × %d = %d.", l, w, l*w))
		}},
		Template{ge, "3.MD.8", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 12), between(r, 2, 9)
			return numeric(fmt.Sprintf("A rectangular garden is %d feet long and %d feet wide. How many feet of fence go all the way around it?", l, w), itoa(2*(l+w)), fmt.Sprintf("Perimeter = %d + %d + %d + %d = %d.", l, w, l, w, 2*(l+w)))
		}},
		Template{ge, "3.MD.5", 1, func(r *rand.Rand) Item {
			n := between(r, 6, 24)
			return numeric(fmt.Sprintf("A shape is covered by %d unit squares with no gaps or overlaps. What is its area in square units?", n), itoa(n), "Area is the number of unit squares that cover the shape.")
		}},
		Template{ge, "3.G.1", 1, func(r *rand.Rand) Item {
			return multi("Which shapes are quadrilaterals?", []string{"square", "rectangle", "triangle", "rhombus", "hexagon"}, []int{0, 1, 3}, "A quadrilateral has exactly four sides.")
		}},
		Template{ge, "3.G.1", 2, func(r *rand.Rand) Item {
			return choices(r, "A shape has 4 sides, all the same length, and 4 right angles. What is it?", "square", []string{"rectangle that is not a square", "triangle", "pentagon"}, "Four equal sides and four right angles make a square.")
		}},
		Template{ge, "3.G.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			return choices(r, fmt.Sprintf("A rectangle is divided into %d parts with equal areas. What fraction of the area is one part?", d), frac(1, d), []string{frac(d, 1), frac(1, d+1), frac(2, d)}, fmt.Sprintf("Each of %d equal parts is 1/%d of the whole.", d, d))
		}},
		Template{ge, "3.MD.8", 3, func(r *rand.Rand) Item {
			l := between(r, 6, 12)
			w := between(r, 2, 8)
			p := 2 * (l + w)
			return numeric(fmt.Sprintf("A rectangle has a perimeter of %d units and a length of %d units. What is its width in units?", p, l), itoa(w), fmt.Sprintf("Two lengths use %d units, leaving %d for two widths, so each width is %d.", 2*l, p-2*l, w))
		}},
	)
}
