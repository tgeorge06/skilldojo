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
			return choices(r, fmt.Sprintf("There are %d %s. Each one has %d %s inside. Which math sentence (expression) shows how many %s there are in all?", g, pick(r, containers), n, thing, thing),
				fmt.Sprintf("%d × %d", g, n), []string{fmt.Sprintf("%d + %d", g, n), fmt.Sprintf("%d − %d", g+n, n), fmt.Sprintf("%d ÷ %d", g*n, g+1)},
				fmt.Sprintf("%d groups of %d is %d × %d.", g, n, g, n))
		}},
		Template{md, "3.OA.2", 2, func(r *rand.Rand) Item {
			g, n := between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("%s has %d %s and shares them fairly with %d friends. Everyone gets the same amount. How many does each friend get?", pick(r, names), g*n, pick(r, things), g),
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
			each, n := between(r, 200, 450), between(r, 2, 4)
			return numeric(fmt.Sprintf("One bottle holds %d milliliters (mL) of juice. How many milliliters are in %d bottles?", each, n), itoa(each*n), fmt.Sprintf("%d × %d = %d.", each, n, each*n))
		}},
		Template{fr, "3.NF.1", 1, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			return figure(choices(r, fmt.Sprintf("A pizza is cut into %d equal slices. %s eats %d slices. What fraction of the pizza is that?", d, pick(r, names), n),
				frac(n, d), []string{frac(d, n), frac(n, d+1), frac(n, d-1)}, fmt.Sprintf("%d of %d equal parts is %s.", n, d, frac(n, d))), Figure{Kind: "parts", A: d, B: n})
		}},
		Template{fr, "3.NF.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			n := between(r, 1, d-1)
			return figure(choices(r, fmt.Sprintf("This number line goes from 0 to 1. It is cut into %d equal parts. What fraction is the dot on?", d),
				frac(n, d), []string{frac(n, d*2), frac(d, n), frac(n+1, d)}, "Count the equal parts from 0 to the dot."), Figure{Kind: "numberline", A: d, B: n})
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4})
			n := between(r, 1, d-1)
			k := pick(r, []int{2, 3})
			return figure(choices(r, fmt.Sprintf("Which fraction is the same amount as %s? (The same amount is called equivalent.)", frac(n, d)),
				frac(n*k, d*k), []string{frac(n+1, d+1), frac(n*k, d), frac(n, d*k)}, "Multiply the top and bottom by the same number."), Figure{Kind: "parts", A: d, B: n})
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
			d := pick(r, []int{3, 4, 5, 6, 8})
			return choices(r, "Which fraction is the same as 1 whole?", frac(d, d), []string{frac(1, d), frac(d, 1), frac(d-1, d)}, "When the top and bottom numbers match, you have all the parts: 1 whole.")
		}},
		Template{fr, "3.NF.3", 2, func(r *rand.Rand) Item {
			return multi("Which fractions are the same as one half (1/2)?", []string{"2/4", "3/6", "1/3", "4/8", "2/3"}, []int{0, 1, 3}, "2/4, 3/6, and 4/8 are all half.")
		}},
		Template{ge, "3.MD.7", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 9), between(r, 2, 8)
			return figure(numeric(fmt.Sprintf("This rectangle is %d squares long and %d squares wide. How many squares fit inside it? (That is its area.)", l, w), itoa(l*w), fmt.Sprintf("Area = %d × %d = %d squares.", l, w, l*w)), Figure{Kind: "grid", A: l, B: w})
		}},
		Template{ge, "3.MD.8", 2, func(r *rand.Rand) Item {
			l, w := between(r, 3, 12), between(r, 2, 9)
			return figure(numeric(fmt.Sprintf("A garden is shaped like a rectangle. It is %d feet long and %d feet wide. How many feet of fence go all the way around it? (That distance is the perimeter.)", l, w), itoa(2*(l+w)), fmt.Sprintf("Perimeter = %d + %d + %d + %d = %d.", l, w, l, w, 2*(l+w))), Figure{Kind: "rect", A: l, B: w, LabelA: fmt.Sprintf("%d ft", l), LabelB: fmt.Sprintf("%d ft", w)})
		}},
		Template{ge, "3.MD.5", 1, func(r *rand.Rand) Item {
			n := between(r, 6, 24)
			return numeric(fmt.Sprintf("A shape is covered with %d little squares. There are no gaps and none overlap. What is the area of the shape, in squares?", n), itoa(n), "Area is how many squares cover the shape.")
		}},
		Template{ge, "3.G.1", 1, func(r *rand.Rand) Item {
			return multi("Which shapes have exactly 4 sides? (A 4-sided shape is a quadrilateral.)", []string{"square", "rectangle", "triangle", "rhombus", "hexagon"}, []int{0, 1, 3}, "A quadrilateral has exactly four sides.")
		}},
		Template{ge, "3.G.1", 2, func(r *rand.Rand) Item {
			return choices(r, "A shape has 4 sides that are all the same length, and 4 square corners. What is it?", "square", []string{"rectangle that is not a square", "triangle", "pentagon"}, "Four equal sides and four square corners make a square.")
		}},
		Template{ge, "3.G.2", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{2, 3, 4, 6, 8})
			return figure(choices(r, fmt.Sprintf("This rectangle is cut into %d equal parts. What fraction of the rectangle is one part?", d), frac(1, d), []string{frac(d, 1), frac(1, d+1), frac(2, d)}, fmt.Sprintf("Each of %d equal parts is 1/%d of the whole.", d, d)), Figure{Kind: "parts", A: d, B: 1})
		}},
		Template{ge, "3.MD.8", 3, func(r *rand.Rand) Item {
			l := between(r, 5, 9)
			w := between(r, 2, 5)
			p := 2 * (l + w)
			return figure(numeric(fmt.Sprintf("A fence goes all the way around a rectangle-shaped yard. The whole fence is %d feet long. (That is the perimeter.) The long side is %d feet. How long is the short side, in feet?", p, l), itoa(w), fmt.Sprintf("Two long sides use %d feet. %d − %d = %d feet for the two short sides, so each short side is %d.", 2*l, p, 2*l, p-2*l, w)), Figure{Kind: "rect", A: l, B: w, LabelA: fmt.Sprintf("%d ft", l), LabelB: "?"})
		}},
	)
}
