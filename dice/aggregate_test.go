package dice_test

import (
	"math"
	"testing"

	"github.com/dmcbane/dieroller-go/dice"
)

func TestParseAggregateAliases(t *testing.T) {
	cases := map[dice.Aggregate][]string{
		dice.Sum:    {"sum", "total"},
		dice.Avg:    {"avg", "average", "mean"},
		dice.HighOf: {"high", "highest", "max"},
		dice.LowOf:  {"low", "lowest", "min"},
		dice.Median: {"median", "med"},
	}
	for want, names := range cases {
		for _, name := range names {
			if got, ok := dice.ParseAggregate(name); !ok || got != want {
				t.Errorf("ParseAggregate(%q) = %v %v, want %v", name, got, ok, want)
			}
		}
	}
	if got, _ := dice.ParseAggregate("SUM"); got != dice.Sum {
		t.Error("aggregate names are not case-insensitive")
	}
	for _, bad := range []string{"worst", "best", "sums", "s", ""} {
		if _, ok := dice.ParseAggregate(bad); ok {
			t.Errorf("ParseAggregate(%q) succeeded, want failure", bad)
		}
	}
}

func TestAggregateNotation(t *testing.T) {
	want := map[dice.Aggregate]string{
		dice.Sum: "SUM", dice.Avg: "AVG", dice.HighOf: "HIGH",
		dice.LowOf: "LOW", dice.Median: "MEDIAN",
	}
	for kind, notation := range want {
		if got := kind.Notation(); got != notation {
			t.Errorf("%v.Notation() = %q, want %q", kind, got, notation)
		}
	}
	// Every canonical name parses back to its kind.
	for _, name := range dice.AggregateNames() {
		kind, ok := dice.ParseAggregate(name)
		if !ok {
			t.Fatalf("canonical name %q does not parse", name)
		}
		if got := kind.Notation(); got != upper(name) {
			t.Errorf("%q renders as %q", name, got)
		}
	}
}

func upper(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'a' && c <= 'z' {
			out[i] = c - 32
		}
	}
	return string(out)
}

func TestAggregateReduce(t *testing.T) {
	totals := []int{14, 12, 3}
	cases := []struct {
		kind     dice.Aggregate
		num, den int
	}{
		{dice.Sum, 29, 1},
		{dice.Avg, 29, 3},
		{dice.HighOf, 14, 1},
		{dice.LowOf, 3, 1},
		{dice.Median, 12, 1},
	}
	for _, c := range cases {
		num, den := c.kind.Reduce(totals)
		if num != c.num || den != c.den {
			t.Errorf("%v.Reduce(%v) = %d/%d, want %d/%d", c.kind, totals, num, den, c.num, c.den)
		}
	}
	// An even number of rolls splits the difference.
	if num, den := dice.Median.Reduce([]int{14, 12, 3, 1}); num != 15 || den != 2 {
		t.Errorf("median of four = %d/%d, want 15/2", num, den)
	}
}

func TestAggregateIgnoresOrder(t *testing.T) {
	for _, kind := range []dice.Aggregate{dice.Sum, dice.Avg, dice.HighOf, dice.LowOf, dice.Median} {
		a := kind.Format([]int{5, 1, 9, 3})
		b := kind.Format([]int{3, 9, 1, 5})
		if a != b {
			t.Errorf("%v depends on roll order: %q vs %q", kind, a, b)
		}
	}
}

func TestAggregateOfOneRoll(t *testing.T) {
	for _, kind := range []dice.Aggregate{dice.Sum, dice.Avg, dice.HighOf, dice.LowOf, dice.Median} {
		if got := kind.Format([]int{13}); got != "13" {
			t.Errorf("%v of a single roll = %q, want %q", kind, got, "13")
		}
	}
}

func TestAggregateLandsBetweenWorstAndBest(t *testing.T) {
	for _, totals := range [][]int{{5, 1, 9, 3}, {-4, -1}, {7}, {2, 2, 2, 2, 2}} {
		low, high := totals[0], totals[0]
		for _, v := range totals {
			low, high = min(low, v), max(high, v)
		}
		for _, kind := range []dice.Aggregate{dice.Avg, dice.HighOf, dice.LowOf, dice.Median} {
			v := kind.Value(totals)
			if v < float64(low) || v > float64(high) {
				t.Errorf("%v of %v = %v, outside [%d, %d]", kind, totals, v, low, high)
			}
		}
	}
}

func TestAggregateFormat(t *testing.T) {
	cases := []struct {
		kind   dice.Aggregate
		totals []int
		want   string
	}{
		// A whole result prints without a decimal point.
		{dice.Sum, []int{40, 31}, "71"},
		{dice.Avg, []int{12, 12}, "12"},
		{dice.Median, []int{11, 13}, "12"},
		// A fractional result is rounded to two places.
		{dice.Avg, []int{14, 12, 3}, "9.67"},
		{dice.Median, []int{14, 11}, "12.5"},
		// A leading zero in the second place is kept.
		{dice.Avg, []int{1, 0, 0, 0}, "0.25"},
		{dice.Avg, []int{101, 100, 100, 100}, "100.25"},
		// A negative aggregate prints the same way.
		{dice.Sum, []int{-4, -1}, "-5"},
		{dice.Median, []int{-14, -11}, "-12.5"},
	}
	for _, c := range cases {
		if got := c.kind.Format(c.totals); got != c.want {
			t.Errorf("%v.Format(%v) = %q, want %q", c.kind, c.totals, got, c.want)
		}
	}
}

// 3/40 is exactly 0.075, but the nearest float64 is a hair below it, so
// rounding a float would show 0.07. Exact arithmetic does not, and neither do
// the Elixir and Racket implementations.
func TestAverageRoundsHalfAwayFromZeroExactly(t *testing.T) {
	forty := func(first int) []int {
		totals := make([]int, 40)
		totals[0] = first
		return totals
	}
	cases := map[int]string{3: "0.08", 7: "0.18", -3: "-0.08"}
	for first, want := range cases {
		if got := dice.Avg.Format(forty(first)); got != want {
			t.Errorf("average of %d over forty rolls = %q, want %q", first, got, want)
		}
	}
	// Rounding a float64 would give the wrong answer here, which is the point.
	if naive := math.Round(3.0/40*100) / 100; naive == 0.08 {
		t.Skip("float64 rounding happens to agree; the exact path is still the one used")
	}
}

func TestDisplayedAverageIsWithinHalfAHundredth(t *testing.T) {
	for n := 1; n <= 24; n++ {
		for sum := -60; sum <= 60; sum++ {
			totals := make([]int, n)
			totals[0] = sum
			shown := dice.Avg.Format(totals)
			exact := float64(sum) / float64(n)
			if math.Abs(parse(t, shown)-exact) > 0.005+1e-9 {
				t.Errorf("%d/%d displayed as %s", sum, n, shown)
			}
		}
	}
}

func parse(t *testing.T, s string) float64 {
	t.Helper()
	var f float64
	var neg bool
	i := 0
	if s[0] == '-' {
		neg, i = true, 1
	}
	whole := 0.0
	for ; i < len(s) && s[i] != '.'; i++ {
		whole = whole*10 + float64(s[i]-'0')
	}
	f = whole
	if i < len(s) {
		scale := 0.1
		for i++; i < len(s); i++ {
			f += float64(s[i]-'0') * scale
			scale /= 10
		}
	}
	if neg {
		return -f
	}
	return f
}
