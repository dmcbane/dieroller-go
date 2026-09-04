// Command pathfinder-character generates Pathfinder ability scores.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dmcbane/dieroller-go/dice"
	"github.com/dmcbane/dieroller-go/internal/version"
	"github.com/dmcbane/dieroller-go/pathfinder"
	"github.com/spf13/cobra"
)

const hint = "Try 'pathfinder-character --help' for more information."

type options struct {
	classic  bool
	standard bool
	heroic   bool
	pool     string
	purchase string
	verbose  bool
	number   int
	asJSON   bool
	seed     int64
	seeded   bool
	version  bool
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
		Use:           "pathfinder-character [flags]",
		Short:         "Generate Pathfinder ability scores",
		Long:          longHelp,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.seeded = cmd.Flags().Changed("seed")
			return run(cmd, args, opts, chosenMethods(cmd))
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&opts.classic, "classic", "c", false, "The classic method: 3D6 per ability.")
	f.BoolVarP(&opts.standard, "standard", "s", false, "The standard method: 4D6 keep high 3 per ability. (this is the default)")
	f.BoolVarP(&opts.heroic, "heroic", "r", false, "The heroic method: 2D6 plus 6 per ability.")
	f.StringVarP(&opts.pool, "pool", "l", "", "The pool method: 24D6 split across the six abilities, minimum 3 each, as 3/3/3/3/3/9.")
	f.StringVarP(&opts.purchase, "purchase", "p", "", "The purchase method: low, standard, high, or epic, granting 10, 15, 20, or 25 points.")
	f.BoolVarP(&opts.verbose, "verbose", "v", false, "Display additional information.")
	f.IntVarP(&opts.number, "number", "n", 1, "Number of characters to roll. Must be greater than 0.")
	f.BoolVarP(&opts.asJSON, "json", "j", false, "Emit one JSON object per character instead of text.")
	f.Int64Var(&opts.seed, "seed", 0, "Seed the random number generator for reproducible characters.")
	f.BoolVarP(&opts.version, "version", "V", false, "Show the version")
	f.SortFlags = false

	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return reworded(err) })
	return cmd
}

// The methods the caller actually named, in declaration order, which is what
// makes selecting two of them an error rather than letting the last one win.
func chosenMethods(cmd *cobra.Command) []string {
	var chosen []string
	for _, name := range []string{"classic", "standard", "heroic", "pool", "purchase"} {
		if cmd.Flags().Changed(name) {
			chosen = append(chosen, name)
		}
	}
	return chosen
}

func run(cmd *cobra.Command, args []string, opts options, chosen []string) error {
	if opts.version {
		cmd.Printf("pathfinder-character %s\n", version.Version)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument %q.", args[0])
	}

	method, err := methodFrom(opts, chosen)
	if err != nil {
		return err
	}

	roller := dice.NewRoller()
	if opts.seeded {
		roller = dice.NewSeededRoller(opts.seed)
	}

	characters, err := pathfinder.NewGenerator(roller).Characters(method, opts.number)
	if err != nil {
		return err
	}

	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	for _, c := range characters {
		fmt.Fprint(out, line(c, method, opts))
	}
	return nil
}

func methodFrom(opts options, chosen []string) (pathfinder.Method, error) {
	if len(chosen) > 1 {
		named := make([]string, len(chosen))
		for i, name := range chosen {
			named[i] = "--" + name
		}
		return pathfinder.Method{}, fmt.Errorf(
			"choose only one generation method (got %s).", strings.Join(named, ", "))
	}

	kind := pathfinder.MethodStandard
	if len(chosen) == 1 {
		switch chosen[0] {
		case "classic":
			kind = pathfinder.MethodClassic
		case "heroic":
			kind = pathfinder.MethodHeroic
		case "pool":
			counts, err := pathfinder.ParsePool(opts.pool)
			if err != nil {
				return pathfinder.Method{}, err
			}
			return pathfinder.Method{Kind: pathfinder.MethodPool, Pool: counts}, nil
		case "purchase":
			campaign, err := pathfinder.ParseCampaign(opts.purchase)
			if err != nil {
				return pathfinder.Method{}, err
			}
			return pathfinder.Method{Kind: pathfinder.MethodPurchase, Points: campaign.Points()}, nil
		}
	}
	return pathfinder.Method{Kind: kind}, nil
}

type characterJSON struct {
	Method     string         `json:"method"`
	Abilities  map[string]int `json:"abilities"`
	Scores     []int          `json:"scores"`
	BonusTotal int            `json:"bonus_total"`
	CostTotal  int            `json:"cost_total"`
}

func line(c pathfinder.Character, m pathfinder.Method, opts options) string {
	switch {
	case opts.asJSON:
		abilities := make(map[string]int, len(c.Abilities))
		for _, pair := range c.Labeled() {
			abilities[pair.Name] = pair.Value
		}
		encoded, err := json.Marshal(characterJSON{
			Method: m.Name(), Abilities: abilities, Scores: c.Abilities,
			BonusTotal: c.BonusTotal, CostTotal: c.CostTotal,
		})
		if err != nil {
			panic(err)
		}
		return string(encoded) + "\n"

	case opts.verbose:
		var b strings.Builder
		for i, pair := range c.Labeled() {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%s: %d", pair.Name, pair.Value)
		}
		fmt.Fprintf(&b, " (bonus %d, cost %d)\n", c.BonusTotal, c.CostTotal)
		return b.String()

	default:
		scores := make([]string, len(c.Abilities))
		for i, v := range c.Abilities {
			scores[i] = strconv.Itoa(v)
		}
		return strings.Join(scores, " ") + "\n"
	}
}
