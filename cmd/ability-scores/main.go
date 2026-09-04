// Command ability-scores writes ability score cost/bonus tables as CSV.
//
// Two files are produced:
//
//   - legal_scores.csv -- the 12,376 spreads buyable with purchase points
//     (scores 7..18)
//   - uniq_scores.csv  -- all 15,890,700 spreads across the full extrapolated
//     table (scores 1..45)
//
// The Racket original also wrote all_scores.csv, enumerating every ordering of
// six scores: 45^6 is roughly 8.3 billion rows, which does not finish in any
// practical time. Every one of those rows is a permutation of a spread already
// present in uniq_scores.csv, and neither cost nor bonus depends on ability
// order, so that file carries no information the unique table lacks. It is
// deliberately not generated.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dmcbane/dieroller-go/internal/version"
	"github.com/dmcbane/dieroller-go/pathfinder"
	"github.com/spf13/cobra"
)

const header = "cost,bonus,s1,s2,s3,s4,s5,s6\n"

func main() {
	var (
		out         string
		legalOnly   bool
		showVersion bool
	)

	cmd := &cobra.Command{
		Use:           "ability-scores [flags]",
		Short:         "Write ability score cost/bonus tables as CSV",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				cmd.Printf("ability-scores %s\n", version.Version)
				return nil
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			if err := write(cmd, filepath.Join(out, "legal_scores.csv"), pathfinder.LegalScores()); err != nil {
				return err
			}
			if legalOnly {
				cmd.Println("Skipped uniq_scores.csv (--legal-only).")
			} else if err := write(cmd, filepath.Join(out, "uniq_scores.csv"), pathfinder.AllScores()); err != nil {
				return err
			}
			cmd.Println("Skipped all_scores.csv: 45^6 is about 8.3 billion rows, one per ordering of")
			cmd.Println("six scores. Cost and bonus do not depend on ability order, so every such row")
			cmd.Println("duplicates a spread already in uniq_scores.csv.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&out, "out", "o", ".", "Directory to write into.")
	cmd.Flags().BoolVar(&legalOnly, "legal-only", false, "Write only legal_scores.csv.")
	cmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Show the version")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Rows are written through a buffered writer as they are enumerated, so memory
// stays flat no matter how large the table is.
func write(cmd *cobra.Command, path string, pool []int) error {
	expected := pathfinder.CountWithRepetition(len(pool), pathfinder.Abilities)
	cmd.Printf("Writing %s (%s rows)...\n", path, group(expected))

	started := time.Now()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	out := bufio.NewWriterSize(file, 1<<20)
	if _, err := out.WriteString(header); err != nil {
		return err
	}

	written := 0
	row := make([]byte, 0, 64)
	for spread := range pathfinder.WithRepetition(pool, pathfinder.Abilities) {
		row = row[:0]
		row = strconv.AppendInt(row, int64(pathfinder.TotalCost(spread)), 10)
		row = append(row, ',')
		row = strconv.AppendInt(row, int64(pathfinder.TotalBonus(spread)), 10)
		for _, score := range spread {
			row = append(row, ',')
			row = strconv.AppendInt(row, int64(score), 10)
		}
		row = append(row, '\n')
		if _, err := out.Write(row); err != nil {
			return err
		}
		written++
	}
	if err := out.Flush(); err != nil {
		return err
	}
	if written != expected {
		return fmt.Errorf("wrote %d rows, expected %d", written, expected)
	}
	cmd.Printf("  done in %.1fs\n", time.Since(started).Seconds())
	return nil
}

// group renders a count with thousands separators.
func group(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
