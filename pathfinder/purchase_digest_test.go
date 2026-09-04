package pathfinder_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/dmcbane/dieroller-go/pathfinder"
)

// The canonical table, dumped from the Racket original itself (Racket 8.16,
// using its own legal-purchase-uniq-sets, ability->cost and
// ability->bonus-points) and reduced to a digest. The Elixir port pins the same
// bytes. This ties this implementation's table to the reference one without
// needing Racket installed or the 12^6 brute force below to run.
func TestPurchaseTableMatchesTheRacketDigest(t *testing.T) {
	var rows []string
	for cost, spreads := range pathfinder.SpreadsByCost() {
		for _, spread := range spreads {
			scores := make([]string, len(spread))
			for i, s := range spread {
				scores[i] = strconv.Itoa(s)
			}
			rows = append(rows, fmt.Sprintf("%d,%d,%s",
				cost, pathfinder.TotalBonus(spread), strings.Join(scores, ",")))
		}
	}
	sort.Strings(rows)

	digest := sha256.New()
	for _, row := range rows {
		digest.Write([]byte(row + "\n"))
	}

	const want = "66d08d1a00f00e84eed3d202aa2e787c74bebd6b0f6e599e5d04a40b9211bcee"
	if got := hex.EncodeToString(digest.Sum(nil)); got != want {
		t.Errorf("purchase table digest = %s, want %s", got, want)
	}
}

// Reproduces the Racket original's 12^6 = 2,985,984 nested-loop brute force
// exactly and asserts it yields the same set the combinatorial enumeration
// does. This is what makes the rewrite safe; it is skipped under -short, as the
// equivalent test is tagged :slow in the Elixir suite.
func TestPurchaseTableEqualsTheBruteForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the 12^6 brute force under -short")
	}

	legal := pathfinder.LegalScores()
	bruteForce := make(map[[6]int]bool)
	var spread [6]int
	for _, str := range legal {
		for _, dex := range legal {
			for _, con := range legal {
				for _, intel := range legal {
					for _, wis := range legal {
						for _, chr := range legal {
							spread = [6]int{str, dex, con, intel, wis, chr}
							sort.Sort(sort.Reverse(sort.IntSlice(spread[:])))
							bruteForce[spread] = true
						}
					}
				}
			}
		}
	}

	enumerated := make(map[[6]int]bool)
	for _, spreads := range pathfinder.SpreadsByCost() {
		for _, s := range spreads {
			enumerated[[6]int{s[0], s[1], s[2], s[3], s[4], s[5]}] = true
		}
	}

	if len(bruteForce) != len(enumerated) {
		t.Fatalf("brute force found %d spreads, enumeration found %d", len(bruteForce), len(enumerated))
	}
	for s := range bruteForce {
		if !enumerated[s] {
			t.Fatalf("the brute force found %v, which the enumeration missed", s)
		}
	}
}
