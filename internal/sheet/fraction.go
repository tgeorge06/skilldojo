package sheet

import (
	"fmt"
	"strconv"
	"strings"
)

// fraction is an exact rational answer; whole numbers have den 1.
type fraction struct {
	num, den int
}

func whole(n int) fraction { return fraction{n, 1} }

func frac(n, d int) fraction {
	g := gcd(abs(n), abs(d))
	if g == 0 {
		g = 1
	}
	if d < 0 {
		n, d = -n, -d
	}
	return fraction{n / g, d / g}
}

func (f fraction) equals(o fraction) bool {
	// Both sides are stored reduced, so compare directly.
	return f.num == o.num && f.den == o.den
}

func (f fraction) String() string {
	if f.den == 1 {
		return strconv.Itoa(f.num)
	}
	return fmt.Sprintf("%d/%d", f.num, f.den)
}

// parseFraction accepts "12", "-4", or "3/4" style answers, reduced or not.
func parseFraction(s string) (fraction, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return fraction{}, fmt.Errorf("empty answer")
	}
	if num, den, found := strings.Cut(s, "/"); found {
		n, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			return fraction{}, fmt.Errorf("bad numerator %q", num)
		}
		d, err := strconv.Atoi(strings.TrimSpace(den))
		if err != nil || d == 0 {
			return fraction{}, fmt.Errorf("bad denominator %q", den)
		}
		return frac(n, d), nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fraction{}, fmt.Errorf("not a number: %q", s)
	}
	return whole(n), nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
