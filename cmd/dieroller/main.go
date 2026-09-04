// Command dieroller rolls dice described in dice notation.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dmcbane/dieroller-go/dice"
	"github.com/dmcbane/dieroller-go/internal/version"
	"github.com/spf13/cobra"
)

const hint = "Try 'dieroller --help' for more information."

// The <dice> <sides> <modifier> <keep> form the notation replaced.
var legacyRx = regexp.MustCompile(`^[-+*]?\d+$`)

type options struct {
	verbose bool
	asJSON  bool
	seed    int64
	seeded  bool
	version bool
}

func main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, hint)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "dieroller [flags] <roll>",
		Short: "Roll dice described in dice notation",
		Long:  longHelp,
		Args:  cobra.ArbitraryArgs,
		// The error is reported by main, with the same hint line and exit
		// status the other implementations use.
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.seeded = cmd.Flags().Changed("seed")
			return run(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show the notation and the dice that were kept.")
	cmd.Flags().BoolVarP(&opts.asJSON, "json", "j", false, "Emit one JSON object per roll instead of text.")
	cmd.Flags().Int64Var(&opts.seed, "seed", 0, "Seed the random number generator for reproducible rolls.")
	cmd.Flags().BoolVarP(&opts.version, "version", "V", false, "Show the version")
	cmd.Flags().SortFlags = false

	// pflag's own wording differs from the other implementations'; this brings
	// an unknown option back in line with them.
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return reworded(err)
	})
	return cmd
}

var unknownFlagRx = regexp.MustCompile(`^unknown (?:shorthand )?flag: -+(\S+)`)

func reworded(err error) error {
	if m := unknownFlagRx.FindStringSubmatch(err.Error()); m != nil {
		dashes := "--"
		if len(m[1]) == 1 {
			dashes = "-"
		}
		return fmt.Errorf("unrecognized option %s%s.", dashes, m[1])
	}
	return err
}

func run(cmd *cobra.Command, args []string, opts options) error {
	if opts.version {
		cmd.Printf("dieroller %s\n", version.Version)
		return nil
	}

	batch, err := batchFrom(args)
	if err != nil {
		return err
	}

	roller := dice.NewRoller()
	if opts.seeded {
		roller = dice.NewSeededRoller(opts.seed)
	}

	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return render(out, roller, batch, opts)
}

func batchFrom(args []string) (dice.Batch, error) {
	switch {
	case len(args) == 0:
		return dice.Batch{}, dice.ErrNoExpression
	// Caught before parsing so the old positional form gets a migration
	// message rather than "expression contains no dice".
	case legacyForm(args):
		return dice.Batch{}, fmt.Errorf("the <dice> <sides> <modifier> <keep> arguments have been"+
			" replaced by dice notation; try: dieroller %s", suggestion(args))
	case len(args) == 1:
		return dice.ParseRoll(args[0])
	default:
		return dice.Batch{}, fmt.Errorf("a roll is one argument; quote the whole expression,"+
			" for example: dieroller %q", strings.Join(args, " "))
	}
}

func legacyForm(args []string) bool {
	if len(args) > 4 {
		return false
	}
	for _, a := range args {
		if !legacyRx.MatchString(a) {
			return false
		}
	}
	return true
}

// Rebuilds the old positional arguments as the equivalent expression.
func suggestion(args []string) string {
	switch len(args) {
	case 1:
		return args[0] + "d20"
	case 2:
		return args[0] + "d" + args[1]
	case 3:
		return args[0] + "d" + args[1] + modifierText(args[2])
	default:
		return args[0] + "d" + args[1] + "k" + args[3] + modifierText(args[2])
	}
}

func modifierText(s string) string {
	op, amount, err := dice.ParseModifier(s)
	if err != nil || (op == dice.Add && amount == 0) {
		return ""
	}
	return fmt.Sprintf("%v%d", op, amount)
}

func render(out *bufio.Writer, roller *dice.Roller, b dice.Batch, opts options) error {
	notation := b.Notation()

	// Without an aggregate each roll is a line of its own, printed as it is
	// made, so a huge repeat count streams rather than buffering. An aggregate
	// is a property of the whole batch and cannot report anything until the
	// last roll is in; verbose still shows the rolls as they happen.
	var each func(dice.Outcome)
	switch {
	case b.Aggregate != dice.NoAggregate && opts.asJSON:
		each = nil
	case b.Aggregate != dice.NoAggregate && opts.verbose:
		expr := b.Expr.Notation()
		each = func(o dice.Outcome) { fmt.Fprint(out, verboseLine(o, expr)) }
	case b.Aggregate != dice.NoAggregate:
		each = nil
	case opts.asJSON:
		each = func(o dice.Outcome) { fmt.Fprint(out, jsonLine(o, notation)) }
	case opts.verbose:
		each = func(o dice.Outcome) { fmt.Fprint(out, verboseLine(o, notation)) }
	default:
		each = func(o dice.Outcome) { fmt.Fprintf(out, "%d\n", o.Total) }
	}

	totals := roller.RollBatch(b, each)
	if b.Aggregate == dice.NoAggregate {
		return nil
	}
	fmt.Fprint(out, summary(b, totals, opts))
	return nil
}

// One parenthesised group per dice term, so a single-group expression reads
// exactly as it always has: "4D6K3 (5 4 4) => 13".
func verboseLine(o dice.Outcome, notation string) string {
	var b strings.Builder
	b.WriteString(notation)
	for _, g := range o.Groups {
		b.WriteString(" (")
		for i, v := range g.Kept {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strconv.Itoa(v))
		}
		b.WriteByte(')')
	}
	fmt.Fprintf(&b, " => %d\n", o.Total)
	return b.String()
}

type groupJSON struct {
	Notation string `json:"notation"`
	Rolled   []int  `json:"rolled"`
	Kept     []int  `json:"kept"`
	Sum      int    `json:"sum"`
}

type rollJSON struct {
	Notation string      `json:"notation"`
	Groups   []groupJSON `json:"groups"`
	Subtotal int         `json:"subtotal"`
	Total    int         `json:"total"`
}

type summaryJSON struct {
	Notation   string  `json:"notation"`
	Aggregate  string  `json:"aggregate"`
	Expression string  `json:"expression"`
	Repeat     int     `json:"repeat"`
	Rolls      []int   `json:"rolls"`
	Value      float64 `json:"value"`
}

func jsonLine(o dice.Outcome, notation string) string {
	groups := make([]groupJSON, len(o.Groups))
	for i, g := range o.Groups {
		groups[i] = groupJSON{g.Spec.Notation(), g.Rolled, g.Kept, g.Total}
	}
	return encode(rollJSON{notation, groups, o.Subtotal, o.Total})
}

func summary(b dice.Batch, totals []int, opts options) string {
	switch {
	case opts.asJSON:
		return encode(summaryJSON{
			Notation:   b.Notation(),
			Aggregate:  strings.ToLower(b.Aggregate.Notation()),
			Expression: b.Expr.Notation(),
			Repeat:     b.Repeat,
			Rolls:      totals,
			Value:      b.Aggregate.Value(totals),
		})
	case opts.verbose:
		return fmt.Sprintf("%s => %s\n", b.Notation(), b.Aggregate.Format(totals))
	default:
		return b.Aggregate.Format(totals) + "\n"
	}
}

// The values here are all finite numbers and plain strings, so encoding cannot
// fail; a panic would be a bug in this file rather than bad input.
func encode(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(encoded) + "\n"
}
