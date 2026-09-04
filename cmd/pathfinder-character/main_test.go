package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/dmcbane/dieroller-go/pathfinder"
)

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
		t.Fatalf("pathfinder-character %v failed: %v", args, err)
	}
	return out
}

func lines(t *testing.T, args ...string) []string {
	t.Helper()
	return strings.Split(strings.TrimSuffix(output(t, args...), "\n"), "\n")
}

func errorOf(t *testing.T, args ...string) string {
	t.Helper()
	if _, err := execute(t, args...); err != nil {
		return err.Error()
	}
	t.Fatalf("pathfinder-character %v succeeded, want an error", args)
	return ""
}

func TestEveryMethodRuns(t *testing.T) {
	for _, args := range [][]string{
		{"-c"}, {"-s"}, {"-r"}, {"-l", "3/3/4/6/4/4"}, {"-p", "epic"}, {},
	} {
		got := lines(t, append(args, "-n", "3", "--seed", "1")...)
		if len(got) != 3 {
			t.Fatalf("%v produced %d lines, want 3", args, len(got))
		}
		for _, line := range got {
			if fields := strings.Fields(line); len(fields) != 6 {
				t.Errorf("%v produced %q, want six scores", args, line)
			}
		}
	}
}

func TestCharactersAreSortedWeakestFirst(t *testing.T) {
	out := lines(t, "-c", "-v", "-n", "20", "--seed", "2")
	previous := -1000
	for _, line := range out {
		start := strings.Index(line, "(bonus ")
		if start < 0 {
			t.Fatalf("verbose line has no bonus: %q", line)
		}
		var bonus int
		if _, err := fmtSscan(line[start:], &bonus); err != nil {
			t.Fatalf("could not read the bonus from %q", line)
		}
		if bonus < previous {
			t.Errorf("characters are not sorted weakest first: %d after %d", bonus, previous)
		}
		previous = bonus
	}
}

func fmtSscan(s string, bonus *int) (int, error) {
	var cost int
	return fmt.Sscanf(s, "(bonus %d, cost %d)", bonus, &cost)
}

func TestVerboseAndPlainFormats(t *testing.T) {
	verbose := output(t, "-s", "-n", "1", "-v", "--seed", "3")
	for _, name := range pathfinder.Abbreviations {
		if !strings.Contains(verbose, name+": ") {
			t.Errorf("verbose output does not label %s: %q", name, verbose)
		}
	}
	if !strings.Contains(verbose, "(bonus ") || !strings.Contains(verbose, ", cost ") {
		t.Errorf("verbose output has no totals: %q", verbose)
	}
	// Plain output is one character per line, six scores each, no labels.
	plain := output(t, "-s", "-n", "4", "--seed", "3")
	if strings.Contains(plain, "STR") {
		t.Errorf("plain output is labelled: %q", plain)
	}
}

func TestJSONOutput(t *testing.T) {
	var c characterJSON
	if err := json.Unmarshal([]byte(output(t, "-p", "epic", "-n", "1", "--json", "--seed", "4")), &c); err != nil {
		t.Fatal(err)
	}
	if c.Method != "purchase 25" || c.CostTotal != 25 || len(c.Scores) != 6 || len(c.Abilities) != 6 {
		t.Errorf("json character = %+v", c)
	}
	if c.Abilities["STR"] != c.Scores[0] {
		t.Error("the labelled abilities disagree with the score list")
	}
	// The method is named for the parameters it carries.
	if err := json.Unmarshal([]byte(output(t, "-l", "3/3/4/6/4/4", "-n", "1", "--json", "--seed", "4")), &c); err != nil {
		t.Fatal(err)
	}
	if c.Method != "pool 3/3/4/6/4/4" {
		t.Errorf("pool method reported as %q", c.Method)
	}
}

func TestPurchaseCharactersSpendTheBudget(t *testing.T) {
	budgets := map[string]int{"low": 10, "standard": 15, "high": 20, "epic": 25}
	for name, want := range budgets {
		for _, line := range lines(t, "-p", name, "-n", "20", "--seed", "5") {
			total := 0
			for _, field := range strings.Fields(line) {
				var score int
				if _, err := fmt.Sscanf(field, "%d", &score); err != nil {
					t.Fatal(err)
				}
				cost, _ := pathfinder.Cost(score)
				total += cost
			}
			if total != want {
				t.Fatalf("a %s character cost %d, want %d", name, total, want)
			}
		}
	}
}

func TestSeeding(t *testing.T) {
	if output(t, "-s", "-n", "5", "--seed", "9") != output(t, "-s", "-n", "5", "--seed", "9") {
		t.Error("the same seed gave different characters")
	}
	if output(t, "-s", "-n", "5", "--seed", "9") == output(t, "-s", "-n", "5", "--seed", "10") {
		t.Error("two different seeds gave the same characters")
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"-p", "banana"}, "purchase type must be one of low, standard, high, or epic."},
		{[]string{"--pool", "3/3/3/3/3"}, "dice per attribute must specify die quantity for six attributes."},
		{[]string{"--pool", "2/3/3/3/3/10"}, "a minimum of 3 dice must be used for each attribute."},
		{[]string{"--pool", "3/3/3/3/3/3"}, "you must specify a total of twenty-four dice for the pool."},
		{[]string{"-n", "0"}, "number of characters must be greater than 0."},
		{[]string{"--bogus"}, "unrecognized option --bogus."},
		{[]string{"extra"}, `unexpected argument "extra".`},
		// The methods are mutually exclusive.
		{[]string{"-c", "-r"}, "choose only one generation method (got --classic, --heroic)."},
	}
	for _, c := range cases {
		if got := errorOf(t, c.args...); got != c.want {
			t.Errorf("%v error = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestInformationalFlags(t *testing.T) {
	if got := output(t, "--version"); !strings.HasPrefix(got, "pathfinder-character ") {
		t.Errorf("--version = %q", got)
	}
	if output(t, "-V") != output(t, "--version") {
		t.Error("-V and --version disagree")
	}
	if got := output(t, "--help"); !strings.Contains(got, "weakest first") {
		t.Errorf("help does not describe the ordering: %q", got)
	}
}
