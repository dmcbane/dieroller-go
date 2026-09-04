package dice

import (
	"sort"
	"strconv"
	"strings"
)

// Aggregate reduces the repeated rolls of one expression to a single number.
//
// 6x4d6k3 produces six independent totals and reports them one by one. An
// aggregate says what to do with the six instead: sum(6x4d6k3) adds them up,
// avg(6x4d6k3) averages them, high(6x4d6k3) reports the best of them.
type Aggregate int

const (
	// NoAggregate is the zero value: report each roll rather than reducing.
	NoAggregate Aggregate = iota
	Sum
	Avg
	HighOf
	LowOf
	Median
)

// Each kind answers to several names, so the notation can read the way the
// roller thinks of it -- max and high are one aggregate. The canonical name is
// first: it is the one Notation renders and the one the error message for an
// unknown aggregate offers.
var aggregateNames = []struct {
	kind  Aggregate
	names []string
}{
	{Sum, []string{"sum", "total"}},
	{Avg, []string{"avg", "average", "mean"}},
	{HighOf, []string{"high", "highest", "max"}},
	{LowOf, []string{"low", "lowest", "min"}},
	{Median, []string{"median", "med"}},
}

// ParseAggregate looks up an aggregate by any of its names, case-insensitively.
func ParseAggregate(name string) (Aggregate, bool) {
	lowered := strings.ToLower(name)
	for _, entry := range aggregateNames {
		for _, candidate := range entry.names {
			if candidate == lowered {
				return entry.kind, true
			}
		}
	}
	return NoAggregate, false
}

// AggregateNames returns the canonical name of every aggregate, in the order
// they are documented.
func AggregateNames() []string {
	names := make([]string, 0, len(aggregateNames))
	for _, entry := range aggregateNames {
		names = append(names, entry.names[0])
	}
	return names
}

// Notation renders an aggregate in canonical notation.
func (a Aggregate) Notation() string {
	for _, entry := range aggregateNames {
		if entry.kind == a {
			return strings.ToUpper(entry.names[0])
		}
	}
	return ""
}

// Reduce returns the exact value of the aggregate over a batch's roll totals,
// as a numerator over a denominator. Sum, HighOf, LowOf and an odd-length
// Median have a denominator of 1; Avg and an even-length Median do not.
//
// Reporting a ratio rather than a float is what lets Format round exactly. It
// is also what --json reports, as a float, since a machine reading it wants the
// unrounded value.
func (a Aggregate) Reduce(totals []int) (num, den int) {
	switch a {
	case Sum:
		sum := 0
		for _, t := range totals {
			sum += t
		}
		return sum, 1
	case Avg:
		sum := 0
		for _, t := range totals {
			sum += t
		}
		return sum, len(totals)
	case HighOf:
		best := totals[0]
		for _, t := range totals {
			if t > best {
				best = t
			}
		}
		return best, 1
	case LowOf:
		worst := totals[0]
		for _, t := range totals {
			if t < worst {
				worst = t
			}
		}
		return worst, 1
	case Median:
		sorted := make([]int, len(totals))
		copy(sorted, totals)
		sort.Ints(sorted)
		middle := len(sorted) / 2
		if len(sorted)%2 == 1 {
			return sorted[middle], 1
		}
		return sorted[middle-1] + sorted[middle], 2
	}
	return 0, 1
}

// Value returns the aggregate as a float, which is what --json reports.
func (a Aggregate) Value(totals []int) float64 {
	num, den := a.Reduce(totals)
	return float64(num) / float64(den)
}

// Format renders the aggregate of totals for display.
//
// A fractional result is rounded to two places, which keeps an average legible
// without claiming a precision the dice do not have. A result that is whole
// after that rounding prints as a whole number: the median of six rolls is
// often exactly 12, and 12.0 reads like a defect.
//
// The rounding is done on an exact count of hundredths rather than on a float.
// The average of forty rolls totalling three is exactly 0.075, which rounds to
// 0.08 -- but the nearest float64 to 0.075 is a hair below it, so rounding a
// float would give 0.07. The Elixir and Racket implementations round exactly
// for the same reason, so all three agree on every value.
func (a Aggregate) Format(totals []int) string {
	num, den := a.Reduce(totals)
	return hundredthsString(roundHundredths(num, den))
}

// roundHundredths returns num/den scaled by 100 and rounded half away from
// zero, which is how a person rounds by hand.
func roundHundredths(num, den int) int {
	if num >= 0 {
		return (num*200 + den) / (den * 2)
	}
	return -((-num*200 + den) / (den * 2))
}

func hundredthsString(h int) string {
	if h%100 == 0 {
		return strconv.Itoa(h / 100)
	}
	sign := ""
	if h < 0 {
		sign, h = "-", -h
	}
	// 100 plus the remainder, with the leading 1 sliced off, is the remainder
	// zero-padded to two digits: 8 becomes "08" and 50 becomes "50".
	places := strings.TrimRight(strconv.Itoa(100 + h%100)[1:], "0")
	return sign + strconv.Itoa(h/100) + "." + places
}
