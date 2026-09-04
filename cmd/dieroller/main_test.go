package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Drives the command the way main does, so the tests exercise exactly that
// path rather than a separate string-building one.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func output(t *testing.T, args ...string) string {
	t.Helper()
	out, err := execute(t, args...)
	if err != nil {
		t.Fatalf("dieroller %v failed: %v", args, err)
	}
	return out
}

func lines(t *testing.T, args ...string) []string {
	t.Helper()
	return strings.Split(strings.TrimSuffix(output(t, args...), "\n"), "\n")
}

func errorOf(t *testing.T, args ...string) string {
	t.Helper()
	_, err := execute(t, args...)
	if err == nil {
		t.Fatalf("dieroller %v succeeded, want an error", args)
	}
	return err.Error()
}

func TestRolling(t *testing.T) {
	if got := output(t, "3d1+3", "-v"); got != "3D1+3 (1 1 1) => 6\n" {
		t.Errorf("3d1+3 -v = %q", got)
	}
	if got := output(t, "2d1+1d1-1", "-v"); got != "2D1+1D1-1 (1 1) (1) => 2\n" {
		t.Errorf("several groups and constants = %q", got)
	}
	if got := output(t, "2d1-1d1", "-v"); got != "2D1-1D1 (1 1) (1) => 1\n" {
		t.Errorf("a subtracted group = %q", got)
	}
	if got := output(t, "2d1+3*2", "-v"); got != "2D1+3*2 (1 1) => 10\n" {
		t.Errorf("a trailing multiplier = %q", got)
	}
	// dh renders as keep-low, dl as keep-high.
	if got := output(t, "4d1dh1", "-v"); !strings.HasPrefix(got, "4D1KL3 (") {
		t.Errorf("4d1dh1 -v = %q", got)
	}
	// A quoted expression may contain spaces.
	if got := output(t, "2d1 + 1d1", "-v"); !strings.HasPrefix(got, "2D1+1D1 (") {
		t.Errorf("a spaced expression = %q", got)
	}
}

func TestRepeatCount(t *testing.T) {
	if got := lines(t, "6x1d1"); len(got) != 6 {
		t.Errorf("6x1d1 produced %d lines, want 6", len(got))
	}
	// The repeat count is not part of the rendered notation.
	if got := output(t, "3x1d1", "-v"); got != "1D1 (1) => 1\n1D1 (1) => 1\n1D1 (1) => 1\n" {
		t.Errorf("3x1d1 -v = %q", got)
	}
	if got := lines(t, "2X1d1"); len(got) != 2 {
		t.Error("the repeat count is not case-insensitive")
	}
	if got := errorOf(t, "0x3d6"); got != "repeat count must be greater than 0." {
		t.Errorf("0x3d6 error = %q", got)
	}
}

func TestAggregates(t *testing.T) {
	cases := map[string]string{
		"sum(6x1d1)":    "6",
		"avg(6x1d1)":    "1",
		"high(6x1d1)":   "1",
		"low(6x1d1)":    "1",
		"median(6x1d1)": "1",
		"sum(3d1)":      "3",
		"avg(2x1d1+1)":  "2",
	}
	for in, want := range cases {
		if got := output(t, in); got != want+"\n" {
			t.Errorf("%s = %q, want %q", in, got, want+"\n")
		}
	}
	// The colon form means the same as the parenthesised one.
	if output(t, "sum:6x1d1") != output(t, "sum(6x1d1)") {
		t.Error("the colon and parenthesised forms disagree")
	}
	// Verbose shows each roll and then the summary.
	want := "1D1 (1) => 1\n1D1 (1) => 1\n1D1 (1) => 1\nSUM(3x1D1) => 3\n"
	if got := output(t, "sum(3x1d1)", "-v"); got != want {
		t.Errorf("sum(3x1d1) -v = %q, want %q", got, want)
	}
	// Every alias reaches its canonical notation.
	if got := output(t, "max:2x1d1", "-v"); !strings.Contains(got, "HIGH(2x1D1) => 1") {
		t.Errorf("max:2x1d1 -v = %q", got)
	}
	if got := output(t, "mean:2x1d1", "-v"); !strings.Contains(got, "AVG(2x1D1) => 1") {
		t.Errorf("mean:2x1d1 -v = %q", got)
	}
}

func TestAggregateErrors(t *testing.T) {
	cases := map[string]string{
		"worst(6x4d6k3)": `unknown aggregate "worst"; use sum, avg, high, low, or median.`,
		"sum(0x1d6)":     "repeat count must be greater than 0.",
		"sum(3x2d6k5)":   "dice must be greater than or equal to keep.",
		"sum(3x2)":       `expression contains no dice: "sum(3x2)"`,
	}
	for in, want := range cases {
		if got := errorOf(t, in); got != want {
			t.Errorf("%s error = %q, want %q", in, got, want)
		}
	}
}

func TestOutputFormats(t *testing.T) {
	if got := output(t, "3d1"); got != "3\n" {
		t.Errorf("plain output = %q", got)
	}

	var roll rollJSON
	if err := json.Unmarshal([]byte(output(t, "2d1", "--json")), &roll); err != nil {
		t.Fatal(err)
	}
	if roll.Notation != "2D1" || roll.Total != 2 || len(roll.Groups) != 1 {
		t.Errorf("json roll = %+v", roll)
	}
	if roll.Groups[0].Sum != 2 || len(roll.Groups[0].Rolled) != 2 {
		t.Errorf("json group = %+v", roll.Groups[0])
	}

	// Each dice group gets its own object.
	if err := json.Unmarshal([]byte(output(t, "2d1+1d1", "--json")), &roll); err != nil {
		t.Fatal(err)
	}
	if len(roll.Groups) != 2 || roll.Total != 3 {
		t.Errorf("json with two groups = %+v", roll)
	}

	// A repeat count emits one object per line.
	if got := lines(t, "3x2d1", "--json"); len(got) != 3 {
		t.Errorf("json with a repeat count produced %d lines, want 3", len(got))
	}

	// An aggregate emits one object for the whole batch.
	var summary summaryJSON
	if err := json.Unmarshal([]byte(output(t, "sum(3x2d1)", "--json")), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Notation != "SUM(3x2D1)" || summary.Aggregate != "sum" ||
		summary.Expression != "2D1" || summary.Repeat != 3 || summary.Value != 6 ||
		len(summary.Rolls) != 3 {
		t.Errorf("json summary = %+v", summary)
	}
	// JSON wins over verbose, as it does for a plain roll.
	if output(t, "sum(3x1d1)", "--json", "-v") != output(t, "sum(3x1d1)", "--json") {
		t.Error("verbose changed the aggregated json output")
	}
}

func TestSeeding(t *testing.T) {
	if output(t, "10x10d100", "--seed", "42", "-v") != output(t, "10x10d100", "--seed", "42", "-v") {
		t.Error("the same seed gave different output")
	}
	if output(t, "20d1000", "--seed", "1") == output(t, "20d1000", "--seed", "2") {
		t.Error("two different seeds gave the same output")
	}
}

func TestInformationalFlags(t *testing.T) {
	help := output(t, "--help")
	for _, want := range []string{"kl<n>", "6x4d6k3", "<aggregate>"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not mention %q", want)
		}
	}
	if got := output(t, "--version"); !strings.HasPrefix(got, "dieroller ") {
		t.Errorf("--version = %q", got)
	}
	if output(t, "-V") != output(t, "--version") {
		t.Error("-V and --version disagree")
	}
}

func TestErrors(t *testing.T) {
	if got := errorOf(t, "-v"); got != "no dice expression given." {
		t.Errorf("no expression error = %q", got)
	}
	if got := errorOf(t, "2+3"); !strings.Contains(got, "expression contains no dice") {
		t.Errorf("no dice error = %q", got)
	}
	if got := errorOf(t, "--bogus"); got != "unrecognized option --bogus." {
		t.Errorf("unknown option error = %q", got)
	}
	// The flags the notation replaced.
	for _, flag := range []string{"--dice", "--sides", "--keep", "--modifier", "--iterations"} {
		if got := errorOf(t, flag, "3"); !strings.Contains(got, "unrecognized option "+flag) {
			t.Errorf("%s error = %q", flag, got)
		}
	}
	// The old positional form suggests its notation equivalent.
	suggestions := []struct {
		args []string
		want string
	}{
		{[]string{"5"}, "try: dieroller 5d20"},
		{[]string{"3", "6"}, "try: dieroller 3d6"},
		{[]string{"3", "6", "+3"}, "try: dieroller 3d6+3"},
		{[]string{"3", "6", "+6", "2"}, "try: dieroller 3d6k2+6"},
	}
	for _, c := range suggestions {
		if got := errorOf(t, c.args...); !strings.Contains(got, c.want) {
			t.Errorf("%v error = %q, want one containing %q", c.args, got, c.want)
		}
	}
	if got := errorOf(t, "2d6", "+", "1d8"); !strings.Contains(got, "quote the whole expression") {
		t.Errorf("unquoted expression error = %q", got)
	}
	if got := errorOf(t, "4d6k3", "extra"); !strings.Contains(got, "one argument") {
		t.Errorf("trailing junk error = %q", got)
	}
}
