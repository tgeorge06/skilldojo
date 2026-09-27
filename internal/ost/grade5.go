package ost

import (
	"fmt"
	"math/rand/v2"
)

// Grade 5 templates. Categories: Fractions (5.NF, 5.MD.2), Decimals (5.NBT,
// 5.OA.1-2, 5.MD.1), Geometry (5.G, 5.MD.3-5, 5.OA.3).
func init() {
	const fr, de, ge = "Fractions", "Decimals", "Geometry"
	register(5,
		Template{fr, "5.NF.1", 2, func(r *rand.Rand) Item {
			d1, d2 := pick(r, []int{2, 3, 4}), pick(r, []int{3, 5, 6})
			for d2 == d1 {
				d2 = pick(r, []int{3, 5, 6})
			}
			n1, n2 := between(r, 1, d1-1), between(r, 1, d2-1)
			return numeric(fmt.Sprintf("%s + %s = ?  (Give your answer as a fraction or mixed number.)", frac(n1, d1), frac(n2, d2)), simplify(n1*d2+n2*d1, d1*d2), "Rewrite both fractions with a common denominator, then add.")
		}},
		Template{fr, "5.NF.1", 2, func(r *rand.Rand) Item {
			d1, d2 := 2, pick(r, []int{3, 5, 8})
			n2 := between(r, 1, d2/2)
			return numeric(fmt.Sprintf("%s − %s = ?  (Give your answer as a fraction.)", frac(1, d1), frac(n2, d2)), simplify(d2-2*n2, 2*d2), "Use a common denominator, then subtract the numerators.")
		}},
		Template{fr, "5.NF.2", 3, func(r *rand.Rand) Item {
			return numeric(fmt.Sprintf("%s used 3/4 cup of flour for muffins and 2/3 cup for bread. How many cups of flour did %s use in all? Give a mixed number.", "Jonah", "he"), "1 5/12", "3/4 = 9/12 and 2/3 = 8/12; 9/12 + 8/12 = 17/12 = 1 5/12.")
		}},
		Template{fr, "5.NF.3", 2, func(r *rand.Rand) Item {
			n, d := between(r, 2, 9), between(r, 3, 6)
			for n%d == 0 {
				n = between(r, 2, 9)
			}
			return choices(r, fmt.Sprintf("%d friends share %d sandwiches equally. How much does each friend get?", d, n), simplify(n, d), []string{simplify(d, n), frac(n, d+1), frac(n-1, d)}, fmt.Sprintf("%d ÷ %d = %s.", n, d, frac(n, d)))
		}},
		Template{fr, "5.NF.4", 2, func(r *rand.Rand) Item {
			a, b, c, d := between(r, 1, 3), between(r, 4, 5), between(r, 1, 3), between(r, 4, 6)
			return numeric(fmt.Sprintf("%s × %s = ?  (Give your answer as a fraction.)", frac(a, b), frac(c, d)), simplify(a*c, b*d), "Multiply numerators and multiply denominators.")
		}},
		Template{fr, "5.NF.4", 2, func(r *rand.Rand) Item {
			n, d, k := between(r, 1, 3), pick(r, []int{4, 5, 6, 8}), between(r, 12, 40)
			for (k*n)%d != 0 {
				k = between(r, 12, 40)
			}
			return numeric(fmt.Sprintf("A class of %d students voted. %s of them chose pizza. How many students chose pizza?", k, frac(n, d)), itoa(k*n/d), fmt.Sprintf("%s × %d = %d.", frac(n, d), k, k*n/d))
		}},
		Template{fr, "5.NF.5", 2, func(r *rand.Rand) Item {
			return choices(r, "Without multiplying, which product is less than 3/4?", "3/4 × 2/5", []string{"3/4 × 1", "3/4 × 7/5", "3/4 × 2"}, "Multiplying by a fraction less than 1 makes the product smaller.")
		}},
		Template{fr, "5.NF.6", 3, func(r *rand.Rand) Item {
			whole, n, d := between(r, 2, 4), 1, pick(r, []int{2, 4})
			cups := whole*d + n
			batches := between(r, 2, 3)
			return numeric(fmt.Sprintf("A recipe needs %d %s cups of flour. %s makes %d batches. How many cups of flour is that? Give a mixed number or fraction.", whole, frac(n, d), "Mia", batches), simplify(cups*batches, d), "Multiply the mixed number by the number of batches.")
		}},
		Template{fr, "5.NF.7", 2, func(r *rand.Rand) Item {
			d, k := pick(r, []int{2, 3, 4, 5}), between(r, 2, 6)
			return numeric(fmt.Sprintf("%s ÷ %d = ?  (Give your answer as a fraction.)", frac(1, d), k), frac(1, d*k), fmt.Sprintf("Dividing 1/%d into %d equal parts gives 1/%d.", d, k, d*k))
		}},
		Template{fr, "5.NF.7", 2, func(r *rand.Rand) Item {
			w, d := between(r, 2, 6), pick(r, []int{2, 3, 4})
			return numeric(fmt.Sprintf("How many %s-cup servings are in %d cups of juice?", frac(1, d), w), itoa(w*d), fmt.Sprintf("%d ÷ 1/%d = %d × %d = %d.", w, d, w, d, w*d))
		}},
		Template{fr, "5.MD.2", 2, func(r *rand.Rand) Item {
			return choices(r, "A line plot shows ribbon lengths: three at 1/2 yard, two at 3/4 yard, and one at 1 yard. What is the total length of ribbon?", "4 yards", []string{"3 yards", "3 1/2 yards", "4 1/2 yards"}, "3 × 1/2 + 2 × 3/4 + 1 = 1 1/2 + 1 1/2 + 1 = 4.")
		}},
		Template{fr, "5.NF.1", 2, func(r *rand.Rand) Item {
			return multi("Which expressions are equal to 1 1/2?", []string{"3/4 + 3/4", "1/2 + 1", "2/3 + 5/6", "1/4 + 1/4", "6/4"}, []int{0, 1, 2, 4}, "Each of those sums or fractions equals 3/2.")
		}},
		Template{de, "5.NBT.1", 2, func(r *rand.Rand) Item {
			return choices(r, "In 4.44, the 4 in the tenths place is worth how much compared with the 4 in the hundredths place?", "10 times as much", []string{"100 times as much", "1/10 as much", "the same"}, "Each place to the left is worth 10 times as much.")
		}},
		Template{de, "5.NBT.2", 1, func(r *rand.Rand) Item {
			v := float64(between(r, 12, 98)) / 10
			p := pick(r, []int{10, 100, 1000})
			return numeric(fmt.Sprintf("%s × %d = ?", dec(v, 1), p), dec(v*float64(p), 0), "Multiplying by a power of ten moves the decimal point to the right.")
		}},
		Template{de, "5.NBT.3", 1, func(r *rand.Rand) Item {
			v := float64(between(r, 101, 999)) / 100
			return choices(r, fmt.Sprintf("Which shows %s in expanded form?", dec(v, 2)), expanded(v), []string{expandedWrong(v, 1), expandedWrong(v, 2), expandedWrong(v, 3)}, "Write each digit times its place value.")
		}},
		Template{de, "5.NBT.3", 2, func(r *rand.Rand) Item {
			a := float64(between(r, 100, 899)) / 100
			b := a + float64(between(r, 1, 9))/1000
			return choices(r, "Which comparison is true?", fmt.Sprintf("%s < %s", dec(a, 2), dec(b, 3)), []string{fmt.Sprintf("%s > %s", dec(a, 2), dec(b, 3)), fmt.Sprintf("%s = %s", dec(a, 2), dec(b, 3)), fmt.Sprintf("%s < %s", dec(b, 3), dec(a, 2))}, "Line up the decimal points and compare place by place.")
		}},
		Template{de, "5.NBT.4", 1, func(r *rand.Rand) Item {
			v := float64(between(r, 1005, 9995)) / 1000
			return numeric(fmt.Sprintf("Round %s to the nearest hundredth.", dec(v, 3)), dec(roundTo(v, 2), 2), "Look at the thousandths digit.")
		}},
		Template{de, "5.NBT.5", 2, func(r *rand.Rand) Item {
			a, b := between(r, 123, 987), between(r, 12, 49)
			return numeric(fmt.Sprintf("%d × %d = ?", a, b), itoa(a*b), "Use the standard algorithm.")
		}},
		Template{de, "5.NBT.6", 2, func(r *rand.Rand) Item {
			d, q := between(r, 12, 30), between(r, 20, 90)
			return numeric(fmt.Sprintf("%d ÷ %d = ?", d*q, d), itoa(q), fmt.Sprintf("%d × %d = %d.", d, q, d*q))
		}},
		Template{de, "5.NBT.7", 2, func(r *rand.Rand) Item {
			a, b := float64(between(r, 125, 899))/100, float64(between(r, 105, 699))/100
			return numeric(fmt.Sprintf("%s + %s = ?", dec(a, 2), dec(b, 2)), dec(a+b, 2), "Line up the decimal points, then add.")
		}},
		Template{de, "5.NBT.7", 2, func(r *rand.Rand) Item {
			price, n := float64(between(r, 125, 499))/100, between(r, 3, 8)
			return numeric(fmt.Sprintf("A notebook costs $%s. How much do %d notebooks cost, in dollars?", dec(price, 2), n), dec(price*float64(n), 2), "Multiply the price by the number of notebooks.")
		}},
		Template{de, "5.NBT.7", 3, func(r *rand.Rand) Item {
			total, paid := float64(between(r, 1250, 4999))/100, 50.0
			return numeric(fmt.Sprintf("%s pays for a $%s item with a $50 bill. How much change does %s get, in dollars?", "Theo", dec(total, 2), "he"), dec(paid-total, 2), "Subtract the price from 50.00.")
		}},
		Template{de, "5.OA.1", 2, func(r *rand.Rand) Item {
			a, b, c := between(r, 2, 9), between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("(%d + %d) × %d = ?", a, b, c), itoa((a+b)*c), "Do the operation in parentheses first.")
		}},
		Template{de, "5.OA.2", 2, func(r *rand.Rand) Item {
			a, b := between(r, 3, 9), between(r, 3, 9)
			return choices(r, fmt.Sprintf("Which expression means \"add %d and %d, then multiply by 2\"?", a, b), fmt.Sprintf("2 × (%d + %d)", a, b), []string{fmt.Sprintf("2 × %d + %d", a, b), fmt.Sprintf("(%d + %d) ÷ 2", a, b), fmt.Sprintf("%d + %d × 2", a, b)}, "Parentheses group the addition so it happens first.")
		}},
		Template{de, "5.MD.1", 2, func(r *rand.Rand) Item {
			km := float64(between(r, 12, 95)) / 10
			return numeric(fmt.Sprintf("How many meters are in %s kilometers?", dec(km, 1)), dec(km*1000, 0), "1 kilometer = 1,000 meters.")
		}},
		Template{de, "5.NBT.2", 2, func(r *rand.Rand) Item {
			return multi("Which expressions are equal to 3,400?", []string{"34 × 10²", "3.4 × 10³", "340 × 10", "0.34 × 10²", "34 × 10"}, []int{0, 1, 2}, "10² = 100 and 10³ = 1,000.")
		}},
		Template{ge, "5.G.1", 1, func(r *rand.Rand) Item {
			x, y := between(r, 1, 9), between(r, 1, 9)
			return choices(r, fmt.Sprintf("A point is %d units to the right of the origin and %d units up. What are its coordinates?", x, y), fmt.Sprintf("(%d, %d)", x, y), []string{fmt.Sprintf("(%d, %d)", y, x), fmt.Sprintf("(%d, %d)", x, y+1), fmt.Sprintf("(%d, %d)", x+y, 0)}, "Coordinates are written (x, y): right first, then up.")
		}},
		Template{ge, "5.G.2", 2, func(r *rand.Rand) Item {
			x, y, d := between(r, 1, 6), between(r, 1, 6), between(r, 2, 5)
			return numeric(fmt.Sprintf("Point A is at (%d, %d). Point B is %d units directly above A. What is the y-coordinate of B?", x, y, d), itoa(y+d), "Moving up adds to the y-coordinate.")
		}},
		Template{ge, "5.G.3", 2, func(r *rand.Rand) Item {
			return multi("Which statements about a square are true?", []string{"It is a rectangle.", "It is a rhombus.", "It has exactly one pair of parallel sides.", "All its angles are right angles.", "It is a triangle."}, []int{0, 1, 3}, "A square is both a rectangle and a rhombus, with two pairs of parallel sides.")
		}},
		Template{ge, "5.G.4", 2, func(r *rand.Rand) Item {
			return choices(r, "A quadrilateral has exactly one pair of parallel sides. What is it called?", "trapezoid", []string{"parallelogram", "rhombus", "rectangle"}, "A trapezoid has exactly one pair of parallel sides.")
		}},
		Template{ge, "5.MD.3", 1, func(r *rand.Rand) Item {
			n := between(r, 8, 60)
			return numeric(fmt.Sprintf("A box is packed with %d unit cubes with no gaps or overlaps. What is its volume in cubic units?", n), itoa(n), "Volume is the number of unit cubes that fill the solid.")
		}},
		Template{ge, "5.MD.5", 2, func(r *rand.Rand) Item {
			l, w, h := between(r, 2, 9), between(r, 2, 9), between(r, 2, 9)
			return numeric(fmt.Sprintf("A rectangular prism is %d units long, %d units wide, and %d units tall. What is its volume in cubic units?", l, w, h), itoa(l*w*h), fmt.Sprintf("%d × %d × %d = %d.", l, w, h, l*w*h))
		}},
		Template{ge, "5.MD.5", 3, func(r *rand.Rand) Item {
			l, w, h := between(r, 3, 8), between(r, 2, 6), between(r, 2, 6)
			return numeric(fmt.Sprintf("A box has a volume of %d cubic cm. Its base is %d cm by %d cm. How tall is the box, in cm?", l*w*h, l, w), itoa(h), fmt.Sprintf("%d ÷ (%d × %d) = %d.", l*w*h, l, w, h))
		}},
		Template{ge, "5.MD.4", 2, func(r *rand.Rand) Item {
			a, b := between(r, 2, 5), between(r, 2, 5)
			c := between(r, 2, 4)
			return numeric(fmt.Sprintf("A solid is made of two rectangular prisms: one %d × %d × %d and one %d × %d × %d. What is its total volume in cubic units?", a, b, c, a, b, c+1), itoa(a*b*c+a*b*(c+1)), "Add the volumes of the two prisms.")
		}},
		Template{ge, "5.OA.3", 2, func(r *rand.Rand) Item {
			s1, s2 := between(r, 2, 5), between(r, 2, 5)*2
			return choices(r, fmt.Sprintf("Pattern A adds %d each time starting at 0. Pattern B adds %d each time starting at 0. How does each term of B compare with A?", s1, s2), fmt.Sprintf("B is %d times A", s2/s1), []string{fmt.Sprintf("B is %d more than A", s2-s1), "B is half of A", "They are equal"}, fmt.Sprintf("%d is %d times %d, so every term of B is %d times the matching term of A.", s2, s2/s1, s1, s2/s1))
		}},
	)
}

// expanded writes a decimal like 3.47 as 3 + 0.4 + 0.07.
func expanded(v float64) string {
	s := dec(v, 2)
	whole := s[:len(s)-3]
	t, h := s[len(s)-2:len(s)-1], s[len(s)-1:]
	out := whole
	if t != "0" {
		out += " + 0." + t
	}
	if h != "0" {
		out += " + 0.0" + h
	}
	return out
}

func expandedWrong(v float64, kind int) string {
	s := dec(v, 2)
	whole := s[:len(s)-3]
	t, h := s[len(s)-2:len(s)-1], s[len(s)-1:]
	switch kind {
	case 1:
		return whole + " + " + t + " + " + h
	case 2:
		return whole + " + 0.0" + t + " + 0." + h
	default:
		return whole + " + 0." + t + " + 0." + h
	}
}

func roundTo(v float64, places int) float64 {
	p := 1.0
	for i := 0; i < places; i++ {
		p *= 10
	}
	return float64(int64(v*p+0.5)) / p
}
