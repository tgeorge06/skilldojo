package sheet

import (
	"fmt"
	mrand "math/rand/v2"
)

// genQuestion builds one question for op at belt difficulty.
// All answers are non-negative; division always divides evenly.
func genQuestion(op, belt string) Question {
	switch op {
	case OpAddSub:
		return genAddSub(belt)
	case OpMul:
		return genMul(belt)
	case OpDiv:
		return genDiv(belt)
	case OpFrac:
		return genFrac(belt)
	}
	panic("unreachable: op validated by Generate")
}

// randRange returns a uniform int in [lo, hi].
func randRange(lo, hi int) int { return lo + mrand.IntN(hi-lo+1) }

func genAddSub(belt string) Question {
	var lo, hi int
	switch belt {
	case BeltWhite:
		lo, hi = 10, 99 // 2-digit
	case BeltYellow:
		lo, hi = 100, 999 // 3-digit
	default:
		lo, hi = 100, 9999 // up to 4-digit
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

func genMul(belt string) Question {
	var a, b int
	switch belt {
	case BeltWhite:
		// easy tables: 1–5 and 10
		easy := []int{1, 2, 3, 4, 5, 10}
		a, b = easy[mrand.IntN(len(easy))], randRange(1, 10)
	case BeltYellow:
		a, b = randRange(2, 12), randRange(2, 12)
	default:
		a, b = randRange(6, 12), randRange(6, 12) // the hard corner of the table
	}
	return Question{Prompt: fmt.Sprintf("%d × %d", a, b), Op: OpMul, answer: whole(a * b)}
}

func genDiv(belt string) Question {
	// Built backwards from quotient × divisor so it always divides evenly.
	var divisor, quotient int
	switch belt {
	case BeltWhite:
		divisor, quotient = randRange(2, 5), randRange(1, 10)
	case BeltYellow:
		divisor, quotient = randRange(2, 12), randRange(2, 12)
	default:
		divisor, quotient = randRange(3, 12), randRange(5, 25)
	}
	return Question{Prompt: fmt.Sprintf("%d ÷ %d", divisor*quotient, divisor), Op: OpDiv, answer: whole(quotient)}
}

func genFrac(belt string) Question {
	var den int
	switch belt {
	case BeltWhite:
		den = []int{2, 3, 4}[mrand.IntN(3)]
	case BeltYellow:
		den = randRange(3, 8)
	default:
		den = randRange(5, 12)
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
