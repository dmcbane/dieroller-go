package dice_test

import (
	"strings"
	"testing"

	"github.com/dmcbane/dieroller-go/dice"
)

// Notation round-trips, so rendering what was parsed is the clearest assertion.
func rendered(t *testing.T, text string) string {
	t.Helper()
	expr, err := dice.Parse(text)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", text, err)
	}
	return expr.Notation()
}

func onlySpec(t *testing.T, text string) dice.Spec {
	t.Helper()
	expr, err := dice.Parse(text)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", text, err)
	}
	specs := expr.Specs()
	if len(specs) != 1 {
		t.Fatalf("Parse(%q) produced %d specs, want 1", text, len(specs))
	}
	return specs[0]
}

func batch(t *testing.T, text string) dice.Batch {
	t.Helper()
	b, err := dice.ParseRoll(text)
	if err != nil {
		t.Fatalf("ParseRoll(%q) failed: %v", text, err)
	}
	return b
}

func TestParseSingleGroup(t *testing.T) {
	cases := map[string]string{
		"4d6k3+2": "4D6K3+2",
		"3d6":     "3D6",
		"1d20+4":  "1D20+4",
		"1d20-1":  "1D20-1",
		"1d20*2":  "1D20*2",
		// Whitespace is tolerated anywhere.
		"  3d6 + 2  ": "3D6+2",
		"2d6 + 1d8":   "2D6+1D8",
	}
	for in, want := range cases {
		if got := rendered(t, in); got != want {
			t.Errorf("rendered(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseIsCaseInsensitive(t *testing.T) {
	if rendered(t, "4D6K3") != rendered(t, "4d6k3") {
		t.Error("4D6K3 and 4d6k3 render differently")
	}
	if rendered(t, "2D20KL1") != rendered(t, "2d20kl1") {
		t.Error("2D20KL1 and 2d20kl1 render differently")
	}
}

func TestKeepDefaultsToEveryDie(t *testing.T) {
	spec := onlySpec(t, "3d6")
	if spec.Keep != 3 || spec.From != dice.High {
		t.Errorf("3d6 kept %d from %v, want 3 from High", spec.Keep, spec.From)
	}
}

func TestSelectors(t *testing.T) {
	cases := []struct {
		in       string
		keep     int
		from     dice.From
		notation string
	}{
		{"4d6k3", 3, dice.High, "4D6K3"},
		{"4d6kh3", 3, dice.High, "4D6K3"},
		// kl keeps the lowest, which is how disadvantage is written.
		{"2d20kl1", 1, dice.Low, "2D20KL1"},
		// dl drops the lowest, which is keeping the highest of the rest.
		{"4d6dl1", 3, dice.High, "4D6K3"},
		// dh drops the highest, which is keeping the lowest of the rest.
		{"4d6dh1", 3, dice.Low, "4D6KL3"},
	}
	for _, c := range cases {
		spec := onlySpec(t, c.in)
		if spec.Keep != c.keep || spec.From != c.from {
			t.Errorf("%s kept %d from %v, want %d from %v", c.in, spec.Keep, spec.From, c.keep, c.from)
		}
		if got := rendered(t, c.in); got != c.notation {
			t.Errorf("rendered(%q) = %q, want %q", c.in, got, c.notation)
		}
	}
}

func TestKeepingEveryDieRendersWithoutAKeepClause(t *testing.T) {
	for _, in := range []string{"4d6k4", "4d6kl4"} {
		if got := rendered(t, in); got != "4D6" {
			t.Errorf("rendered(%q) = %q, want %q", in, got, "4D6")
		}
	}
}

func TestMultipleGroups(t *testing.T) {
	cases := map[string]string{
		"2d6+1d8":       "2D6+1D8",
		"2d6-1d4":       "2D6-1D4",
		"2d6+1d8-1":     "2D6+1D8-1",
		"1d20+2d6+3":    "1D20+2D6+3",
		"4d6k3+2d20kl1": "4D6K3+2D20KL1",
		"2d6+3*2":       "2D6+3*2",
		// A zero multiplier is a real expression, not an absent one.
		"3d6*0": "3D6*0",
	}
	for in, want := range cases {
		if got := rendered(t, in); got != want {
			t.Errorf("rendered(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseRejection(t *testing.T) {
	for _, bad := range []string{"d20", "4d", "4x6", "", "4d6k", "abc", "2d6+", "+", "4d6kx3", "4d6d1"} {
		if _, err := dice.Parse(bad); err == nil {
			t.Errorf("Parse(%q) succeeded, want an error", bad)
		}
	}
}

// The messages and the order the checks run in are shared with the Elixir and
// Racket implementations.
func TestValidationMessages(t *testing.T) {
	cases := map[string]string{
		"2d6k5":  "dice must be greater than or equal to keep.",
		"2d0":    "sides must be greater than 0.",
		"0d6":    "dice must be greater than 0.",
		"4d6dl4": "keep must be greater than 0.",
	}
	for in, want := range cases {
		_, err := dice.Parse(in)
		if err == nil || err.Error() != want {
			t.Errorf("Parse(%q) error = %v, want %q", in, err, want)
		}
	}
}

func TestMultiplierMustComeLast(t *testing.T) {
	_, err := dice.Parse("2d6*2+3")
	if err == nil || !strings.Contains(err.Error(), "must come last") {
		t.Errorf("Parse(2d6*2+3) error = %v, want one mentioning \"must come last\"", err)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, dice_ := range []int{1, 4, 20, 50} {
		for _, sides := range []int{1, 6, 20, 100} {
			for _, keep := range []int{1, dice_} {
				for _, dir := range []string{"", "h", "l"} {
					for _, mod := range []string{"", "+3", "-1", "*2", "*0"} {
						keepPart := ""
						if keep != dice_ {
							keepPart = "k" + dir + itoa(keep)
						}
						text := itoa(dice_) + "d" + itoa(sides) + keepPart + mod
						expr, err := dice.Parse(text)
						if err != nil {
							t.Fatalf("Parse(%q): %v", text, err)
						}
						// Reparsing the rendered form must land on the same expression.
						again, err := dice.Parse(expr.Notation())
						if err != nil {
							t.Fatalf("Parse(%q): %v", expr.Notation(), err)
						}
						if again.Notation() != expr.Notation() {
							t.Errorf("%q rendered %q, which rendered %q", text, expr.Notation(), again.Notation())
						}
					}
				}
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestParseRoll(t *testing.T) {
	if b := batch(t, "6x4d6k3"); b.Repeat != 6 || b.Aggregate != dice.NoAggregate ||
		b.Expr.Notation() != "4D6K3" {
		t.Errorf("6x4d6k3 = %+v", b)
	}
	if b := batch(t, "4d6k3"); b.Repeat != 1 {
		t.Errorf("a roll with no repeat count repeated %d times", b.Repeat)
	}
	if b := batch(t, "sum(6x4d6k3)"); b.Repeat != 6 || b.Aggregate != dice.Sum {
		t.Errorf("sum(6x4d6k3) = %+v", b)
	}
	// The colon form needs no shell quoting and means the same.
	if batch(t, "sum:6x4d6k3").Notation() != batch(t, "sum(6x4d6k3)").Notation() {
		t.Error("the colon and parenthesised forms disagree")
	}
	if batch(t, "SUM ( 6 X 4d6k3 )").Notation() != batch(t, "sum(6x4d6k3)").Notation() {
		t.Error("an aggregate is not case- and whitespace-insensitive")
	}
	if b := batch(t, "avg(4d6k3)"); b.Repeat != 1 || b.Aggregate != dice.Avg {
		t.Errorf("an aggregate without a repeat count = %+v", b)
	}
}

func TestBatchNotation(t *testing.T) {
	cases := map[string]string{
		// Without an aggregate the repeat count is not part of the roll.
		"4d6k3":         "4D6K3",
		"6x4d6k3":       "4D6K3",
		"sum(6x4d6k3)":  "SUM(6x4D6K3)",
		"avg:100x1d20":  "AVG(100x1D20)",
		"sum(4d6k3)":    "SUM(1x4D6K3)",
		"MAX(2X4d6dl1)": "HIGH(2x4D6K3)",
	}
	for in, want := range cases {
		if got := batch(t, in).Notation(); got != want {
			t.Errorf("batch(%q).Notation() = %q, want %q", in, got, want)
		}
	}
}

func TestParseRollErrors(t *testing.T) {
	cases := map[string]string{
		"worst(6x4d6k3)": `unknown aggregate "worst"; use sum, avg, high, low, or median.`,
		"0x3d6":          "repeat count must be greater than 0.",
		"sum(0x3d6)":     "repeat count must be greater than 0.",
		"7":              `expression contains no dice: "7"`,
		"sum(3x2)":       `expression contains no dice: "sum(3x2)"`,
		"sum(3x2d6k5)":   "dice must be greater than or equal to keep.",
		// The aggregate and repeat count are stripped before the expression is
		// parsed, but an error still names the roll the way it was written.
		"sum(4d6":    `could not parse dice notation: "sum(4d6"`,
		"4d6 k":      `could not parse dice notation: "4d6 k"`,
		"6x4d6 k":    `could not parse dice notation: "6x4d6 k"`,
		"sum(4d6 k)": `could not parse dice notation: "sum(4d6 k)"`,
	}
	for in, want := range cases {
		_, err := dice.ParseRoll(in)
		if err == nil || err.Error() != want {
			t.Errorf("ParseRoll(%q) error = %v, want %q", in, err, want)
		}
	}
}

func TestParseModifier(t *testing.T) {
	cases := []struct {
		in     string
		op     dice.Op
		amount int
	}{
		{"+3", dice.Add, 3}, {"-3", dice.Subtract, 3}, {"*3", dice.Multiply, 3},
		{"6", dice.Add, 6}, {"0", dice.Add, 0},
	}
	for _, c := range cases {
		op, amount, err := dice.ParseModifier(c.in)
		if err != nil || op != c.op || amount != c.amount {
			t.Errorf("ParseModifier(%q) = %v %d %v, want %v %d", c.in, op, amount, err, c.op, c.amount)
		}
	}
	for _, bad := range []string{"+x", "", "3.5"} {
		if _, _, err := dice.ParseModifier(bad); err == nil {
			t.Errorf("ParseModifier(%q) succeeded, want an error", bad)
		}
	}
}
