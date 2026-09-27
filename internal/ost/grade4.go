package ost

import (
	"fmt"
	"math/rand/v2"
)

// Grade 4 templates. Categories: Multiplication and Division (4.OA, 4.NBT,
// 4.MD.2-3), Fractions (4.NF, 4.MD.4), Geometry (4.G, 4.MD.1, 4.MD.5-7).
func init() {
	const md, fr, ge = "Multiplication and Division", "Fractions", "Geometry"
	register(4,
		Template{md, "4.OA.1", 2, func(r *rand.Rand) Item {
			k, n := between(r, 3, 9), between(r, 4, 12)
			return choices(r, fmt.Sprintf("%s has %d %s. %s has %d times as many. Which equation shows how many %s has?", "Ava", n, pick(r, things), "Leo", k, "Leo"),
				fmt.Sprintf("%d × %d = %d", k, n, k*n), []string{fmt.Sprintf("%d + %d = %d", k, n, k+n), fmt.Sprintf("%d − %d = %d", n, k, n-k), fmt.Sprintf("%d ÷ %d = %d", k*n, n, k)},
				fmt.Sprintf("\"%d times as many\" means multiply: %d × %d.", k, k, n))
		}},
		Template{md, "4.OA.2", 2, func(r *rand.Rand) Item {
			k, n := between(r, 3, 8), between(r, 4, 12)
			return numeric(fmt.Sprintf("A blue ribbon is %d cm long. A red ribbon is %d times as long. How long is the red ribbon in centimeters?", n, k), itoa(k*n), fmt.Sprintf("%d × %d = %d.", k, n, k*n))
		}},
		Template{md, "4.OA.3", 3, func(r *rand.Rand) Item {
			per, boxes, extra := between(r, 12, 24), between(r, 4, 9), between(r, 5, 20)
			total := per*boxes + extra
			return numeric(fmt.Sprintf("A store has %d %s of %s with %d in each, plus %d loose ones. How many are there in all?", boxes, pick(r, containers), pick(r, things), per, extra), itoa(total), fmt.Sprintf("%d × %d = %d, then + %d = %d.", boxes, per, boxes*per, extra, total))
		}},
		Template{md, "4.OA.3", 3, func(r *rand.Rand) Item {
			kids, per := between(r, 6, 9), between(r, 30, 60)
			total := kids*per + between(r, 1, kids-1)
			return numeric(fmt.Sprintf("%d students share %d %s as equally as possible. How many are left over?", kids, total, pick(r, things)), itoa(total%kids), fmt.Sprintf("%d ÷ %d = %d remainder %d.", total, kids, total/kids, total%kids))
		}},
		Template{md, "4.OA.4", 2, func(r *rand.Rand) Item {
			n := pick(r, []int{12, 18, 20, 24, 28, 30, 36, 40, 42, 45, 48})
			var f []string
			for d := 2; d < n; d++ {
				if n%d == 0 {
					f = append(f, itoa(d))
				}
			}
			correct := f[r.IntN(len(f))]
			var wrong []string
			for d := 2; len(wrong) < 3; d++ {
				if n%d != 0 {
					wrong = append(wrong, itoa(d))
				}
			}
			return choices(r, fmt.Sprintf("Which number is a factor of %d?", n), correct, wrong, fmt.Sprintf("%d divides %d with no remainder.", 1, n))
		}},
		Template{md, "4.OA.4", 1, func(r *rand.Rand) Item {
			return multi("Which numbers are prime?", []string{"7", "9", "13", "15", "2"}, []int{0, 2, 4}, "A prime number has exactly two factors: 1 and itself.")
		}},
		Template{md, "4.OA.5", 2, func(r *rand.Rand) Item {
			start, step := between(r, 2, 9), between(r, 4, 9)
			return numeric(fmt.Sprintf("A pattern follows the rule \"add %d\" and starts at %d. What is the 5th number?", step, start), itoa(start+4*step), fmt.Sprintf("%d, %d, %d, %d, %d.", start, start+step, start+2*step, start+3*step, start+4*step))
		}},
		Template{md, "4.NBT.1", 2, func(r *rand.Rand) Item {
			d := between(r, 2, 9)
			return choices(r, fmt.Sprintf("In the number %d%d%d, how many times greater is the value of the first %d than the value of the last %d?", d, 0, d, d, d),
				"100 times", []string{"10 times", "1,000 times", "2 times"}, "A digit two places to the left is worth 100 times as much.")
		}},
		Template{md, "4.NBT.2", 1, func(r *rand.Rand) Item {
			a := between(r, 10000, 90000)
			b := a + between(r, 100, 900)
			return choices(r, fmt.Sprintf("Which comparison is true?"), fmt.Sprintf("%d < %d", a, b), []string{fmt.Sprintf("%d > %d", a, b), fmt.Sprintf("%d = %d", a, b), fmt.Sprintf("%d < %d", b, a)}, "Compare the digits from left to right.")
		}},
		Template{md, "4.NBT.3", 1, func(r *rand.Rand) Item {
			n := between(r, 12000, 98000)
			rounded := ((n + 500) / 1000) * 1000
			return numeric(fmt.Sprintf("Round %d to the nearest thousand.", n), itoa(rounded), "Look at the hundreds digit.")
		}},
		Template{md, "4.NBT.4", 1, func(r *rand.Rand) Item {
			a, b := between(r, 2000, 9000), between(r, 1000, 8000)
			return numeric(fmt.Sprintf("%d + %d = ?", a, b), itoa(a+b), "Add place by place, regrouping as needed.")
		}},
		Template{md, "4.NBT.4", 2, func(r *rand.Rand) Item {
			a, b := between(r, 5000, 9900), between(r, 1200, 4900)
			return numeric(fmt.Sprintf("A theater has %d seats. %d tickets have been sold. How many seats are still empty?", a, b), itoa(a-b), fmt.Sprintf("%d − %d = %d.", a, b, a-b))
		}},
		Template{md, "4.NBT.5", 2, func(r *rand.Rand) Item {
			a, b := between(r, 21, 89), between(r, 3, 9)
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), fmt.Sprintf("%d × %d = %d.", a, b, a*b))
		}},
		Template{md, "4.NBT.5", 2, func(r *rand.Rand) Item {
			a, b := between(r, 12, 30), between(r, 11, 25)
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), "Multiply by the tens and ones, then add the partial products.")
		}},
		Template{md, "4.NBT.6", 2, func(r *rand.Rand) Item {
			d, q := between(r, 3, 9), between(r, 20, 120)
			return numeric(fmt.Sprintf("%d ÷ %d = ?", d*q, d), itoa(q), fmt.Sprintf("%d × %d = %d.", d, q, d*q))
		}},
		Template{md, "4.MD.2", 3, func(r *rand.Rand) Item {
			dollars, cents, n := between(r, 2, 9), pick(r, []int{25, 50, 75}), between(r, 3, 6)
			total := (dollars*100 + cents) * n
			return numeric(fmt.Sprintf("One ticket costs $%d.%02d. How much do %d tickets cost, in dollars?", dollars, cents, n), fmt.Sprintf("%d.%02d", total/100, total%100), "Multiply the price by the number of tickets.")
		}},
		Template{md, "4.MD.3", 2, func(r *rand.Rand) Item {
			l, w := between(r, 12, 40), between(r, 5, 20)
			return numeric(fmt.Sprintf("A rectangular field is %d meters by %d meters. What is its area in square meters?", l, w), itoa(l*w), fmt.Sprintf("%d × %d = %d.", l, w, l*w))
		}},
		Template{fr, "4.NF.1", 2, func(r *rand.Rand) Item {
			n, d := between(r, 1, 4), between(r, 5, 8)
			k := pick(r, []int{2, 3, 4})
			return choices(r, fmt.Sprintf("Which fraction is equivalent to %s?", frac(n, d)), frac(n*k, d*k), []string{frac(n+k, d+k), frac(n*k, d), frac(n, d*k)}, "Multiply numerator and denominator by the same number.")
		}},
		Template{fr, "4.NF.2", 2, func(r *rand.Rand) Item {
			a, b := frac(3, 4), frac(2, 3)
			if r.IntN(2) == 0 {
				a, b = frac(5, 6), frac(3, 4)
			}
			return choices(r, fmt.Sprintf("Which comparison is true?"), fmt.Sprintf("%s > %s", a, b), []string{fmt.Sprintf("%s < %s", a, b), fmt.Sprintf("%s = %s", a, b), fmt.Sprintf("%s > %s", b, a)}, "Rewrite with a common denominator, then compare numerators.")
		}},
		Template{fr, "4.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{5, 6, 8, 10, 12})
			a, b := between(r, 1, d/2), between(r, 1, d/2)
			return numeric(fmt.Sprintf("%s + %s = ?  (Give your answer as a fraction.)", frac(a, d), frac(b, d)), simplify(a+b, d), "Add the numerators; the denominator stays the same.")
		}},
		Template{fr, "4.NF.3", 2, func(r *rand.Rand) Item {
			d := pick(r, []int{4, 6, 8, 10})
			a, b := between(r, d/2, d-1), between(r, 1, d/2-1)
			return numeric(fmt.Sprintf("%s ran %s of a mile on Monday and %s of a mile on Tuesday. How much farther did %s run on Monday? Give a fraction of a mile.", "Priya", frac(a, d), frac(b, d), "she"), simplify(a-b, d), "Subtract the numerators.")
		}},
		Template{fr, "4.NF.4", 2, func(r *rand.Rand) Item {
			n, d, k := between(r, 1, 3), pick(r, []int{4, 5, 8}), between(r, 2, 5)
			return numeric(fmt.Sprintf("%d × %s = ?  (Give your answer as a fraction or mixed number.)", k, frac(n, d)), simplify(k*n, d), fmt.Sprintf("%d × %d = %d, over %d.", k, n, k*n, d))
		}},
		Template{fr, "4.NF.5", 2, func(r *rand.Rand) Item {
			n := between(r, 1, 9)
			m := between(r, 11, 59)
			return choices(r, fmt.Sprintf("%s + %s = ?", frac(n, 10), frac(m, 100)), frac(n*10+m, 100), []string{frac(n+m, 100), frac(n+m, 110), frac(n*10+m, 10)}, fmt.Sprintf("%s = %s, then add hundredths.", frac(n, 10), frac(n*10, 100)))
		}},
		Template{fr, "4.NF.6", 1, func(r *rand.Rand) Item {
			n := between(r, 1, 99)
			return numeric(fmt.Sprintf("Write %s as a decimal.", frac(n, 100)), dec(float64(n)/100, 2), "Hundredths take two decimal places.")
		}},
		Template{fr, "4.NF.7", 2, func(r *rand.Rand) Item {
			a := float64(between(r, 10, 89)) / 100
			b := a + float64(between(r, 3, 9))/100
			return choices(r, "Which comparison is true?", fmt.Sprintf("%s < %s", dec(a, 2), dec(b, 2)), []string{fmt.Sprintf("%s > %s", dec(a, 2), dec(b, 2)), fmt.Sprintf("%s = %s", dec(a, 2), dec(b, 2)), fmt.Sprintf("%s < %s", dec(b, 2), dec(a, 2))}, "Compare tenths first, then hundredths.")
		}},
		Template{fr, "4.MD.4", 2, func(r *rand.Rand) Item {
			return choices(r, "A line plot shows pencil lengths: two pencils at 3 1/2 in., three at 3 3/4 in., and one at 4 in. How many pencils are longer than 3 1/2 inches?", "4", []string{"3", "5", "6"}, "Count the pencils at 3 3/4 and 4 inches.")
		}},
		Template{fr, "4.NF.3", 3, func(r *rand.Rand) Item {
			return multi("Which sums equal 5/8?", []string{"1/8 + 4/8", "2/8 + 3/8", "3/8 + 3/8", "5/8 + 0/8", "1/4 + 2/8"}, []int{0, 1, 3}, "Add the numerators over 8; 1/4 + 2/8 is 4/8, not 5/8.")
		}},
		Template{ge, "4.MD.1", 1, func(r *rand.Rand) Item {
			m := between(r, 2, 9)
			return numeric(fmt.Sprintf("How many centimeters are in %d meters?", m), itoa(m*100), "1 meter = 100 centimeters.")
		}},
		Template{ge, "4.MD.1", 2, func(r *rand.Rand) Item {
			h := between(r, 2, 6)
			return numeric(fmt.Sprintf("How many minutes are in %d hours?", h), itoa(h*60), "1 hour = 60 minutes.")
		}},
		Template{ge, "4.MD.5", 1, func(r *rand.Rand) Item {
			return choices(r, "An angle that measures exactly 90 degrees is called what?", "a right angle", []string{"an acute angle", "an obtuse angle", "a straight angle"}, "Right angles measure 90°.")
		}},
		Template{ge, "4.MD.6", 1, func(r *rand.Rand) Item {
			a := pick(r, []int{35, 50, 65, 80, 110, 125, 150})
			kind := "acute"
			if a > 90 {
				kind = "obtuse"
			}
			return choices(r, fmt.Sprintf("An angle measures %d degrees. What kind of angle is it?", a), kind, []string{map[bool]string{true: "obtuse", false: "acute"}[kind == "acute"], "right", "straight"}, "Less than 90° is acute; more than 90° and less than 180° is obtuse.")
		}},
		Template{ge, "4.MD.7", 2, func(r *rand.Rand) Item {
			a := between(r, 20, 70)
			return numeric(fmt.Sprintf("Two angles together make a right angle. One measures %d degrees. What is the other, in degrees?", a), itoa(90-a), fmt.Sprintf("90 − %d = %d.", a, 90-a))
		}},
		Template{ge, "4.G.1", 1, func(r *rand.Rand) Item {
			return choices(r, "Two lines that never meet and stay the same distance apart are called what?", "parallel lines", []string{"perpendicular lines", "intersecting lines", "rays"}, "Parallel lines never cross.")
		}},
		Template{ge, "4.G.2", 2, func(r *rand.Rand) Item {
			return multi("Which shapes always have at least one pair of parallel sides?", []string{"square", "trapezoid", "triangle", "rectangle", "circle"}, []int{0, 1, 3}, "Squares and rectangles have two pairs; a trapezoid has at least one.")
		}},
		Template{ge, "4.G.3", 1, func(r *rand.Rand) Item {
			return choices(r, "How many lines of symmetry does a square have?", "4", []string{"1", "2", "8"}, "Two through the midpoints of the sides and two through the corners.")
		}},
		Template{ge, "4.MD.3", 3, func(r *rand.Rand) Item {
			w := between(r, 4, 12)
			l := w + between(r, 2, 10)
			return numeric(fmt.Sprintf("A rectangle has an area of %d square units and a width of %d units. What is its perimeter in units?", l*w, w), itoa(2*(l+w)), fmt.Sprintf("Length = %d ÷ %d = %d; perimeter = 2 × (%d + %d) = %d.", l*w, w, l, l, w, 2*(l+w)))
		}},
	)
}
