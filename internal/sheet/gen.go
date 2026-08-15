package sheet

import (
	"fmt"
	mrand "math/rand/v2"
)

// genQuestion builds one question for op at the given grade level (1–5).
// All answers are non-negative; division always divides evenly.
func genQuestion(op string, grade int) Question {
	switch op {
	case OpAddSub:
		return genAddSub(grade)
	case OpMul:
		return genMul(grade)
	case OpDiv:
		return genDiv(grade)
	case OpFrac:
		return genFrac(grade)
	}
	panic("unreachable: op validated by Generate")
}

// randRange returns a uniform int in [lo, hi].
func randRange(lo, hi int) int { return lo + mrand.IntN(hi-lo+1) }

func genAddSub(grade int) Question {
	// Grade 1 is special: everything stays within 20 (a+b ≤ 20, a−b ≥ 0).
	if grade <= 1 {
		if mrand.IntN(2) == 0 {
			a := randRange(1, 19)
			b := randRange(1, 20-a)
			return Question{Prompt: fmt.Sprintf("%d + %d", a, b), Op: OpAddSub, answer: whole(a + b)}
		}
		a := randRange(2, 20)
		b := randRange(1, a-1)
		return Question{Prompt: fmt.Sprintf("%d − %d", a, b), Op: OpAddSub, answer: whole(a - b)}
	}

	var lo, hi int
	switch grade {
	case 2:
		lo, hi = 10, 99 // within 100
	case 3:
		lo, hi = 100, 999 // 3-digit
	case 4:
		lo, hi = 100, 9999 // up to 4-digit
	default:
		lo, hi = 1000, 99999 // grade 5: up to 5-digit
	}
	a, b := randRange(lo, hi), randRange(lo, hi)
	if mrand.IntN(2) == 0 {
		return Question{Prompt: fmt.Sprintf("%d + %d", a, b), Op: OpAddSub, answer: whole(a + b)}
	}
	if b > a {
		a, b = b, a // keep subtraction non-negative
	}
	return Question{Prompt: fmt.Sprintf("%d − %d", a, b), Op: OpAddSub, answer: whole(a - b)}
}

func genMul(grade int) Question {
	var a, b int
	switch grade {
	case 1:
		// gentle intro: 1, 2, 5, 10 tables up to ×5
		easy := []int{1, 2, 5, 10}
		a, b = easy[mrand.IntN(len(easy))], randRange(1, 5)
	case 2:
		easy := []int{1, 2, 3, 4, 5, 10}
		a, b = easy[mrand.IntN(len(easy))], randRange(1, 10)
	case 3:
		a, b = randRange(2, 12), randRange(2, 12)
	case 4:
		a, b = randRange(6, 12), randRange(6, 12) // the hard corner of the table
	default:
		a, b = randRange(13, 99), randRange(3, 9) // grade 5: 2-digit × 1-digit
	}
	return Question{Prompt: fmt.Sprintf("%d × %d", a, b), Op: OpMul, answer: whole(a * b)}
}

func genDiv(grade int) Question {
	// Built backwards from quotient × divisor so it always divides evenly.
	var divisor, quotient int
	switch grade {
	case 1:
		divisor, quotient = 2, randRange(1, 10) // halving only
	case 2:
		divisor, quotient = randRange(2, 5), randRange(1, 10)
	case 3:
		divisor, quotient = randRange(2, 12), randRange(2, 12)
	case 4:
		divisor, quotient = randRange(3, 12), randRange(5, 25)
	default:
		divisor, quotient = randRange(3, 12), randRange(10, 99) // grade 5
	}
	return Question{Prompt: fmt.Sprintf("%d ÷ %d", divisor*quotient, divisor), Op: OpDiv, answer: whole(quotient)}
}

func genFrac(grade int) Question {
	var den int
	switch grade {
	case 1:
		den = []int{2, 3}[mrand.IntN(2)]
	case 2:
		den = []int{2, 3, 4}[mrand.IntN(3)]
	case 3:
		den = randRange(3, 8)
	case 4:
		den = randRange(5, 12)
	default:
		den = randRange(7, 16) // grade 5
	}
	// Same-denominator add or subtract; answers stay in (0, 1].
	a := randRange(1, den-1)
	if mrand.IntN(2) == 0 {
		b := randRange(1, den-a) // a+b ≤ den
		return Question{
			Prompt: fmt.Sprintf("%d/%d + %d/%d", a, den, b, den),
			Op:     OpFrac,
			answer: frac(a+b, den),
		}
	}
	b := randRange(1, a) // a−b ≥ 0, but avoid 0 for a friendlier answer
	if b == a {
		b = a - 1
	}
	if b == 0 {
		b = 1
		if a == 1 {
			a = 2
			if a >= den { // den was 2: fall back to addition
				return Question{Prompt: fmt.Sprintf("1/%d + 1/%d", den, den), Op: OpFrac, answer: frac(2, den)}
			}
		}
	}
	return Question{
		Prompt: fmt.Sprintf("%d/%d − %d/%d", a, den, b, den),
		Op:     OpFrac,
		answer: frac(a-b, den),
	}
}
