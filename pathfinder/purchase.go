package pathfinder

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// The purchase (point buy) method.
//
// Every legal ability spread -- six scores drawn from 7..18 -- is enumerated
// once and grouped by total cost, so selecting a character is a map lookup plus
// one random index. The Racket original recomputed this with a memoized 12^6
// brute force on first use.

// Campaign is a named purchase budget.
type Campaign int

const (
	Low Campaign = iota
	Standard
	High
	Epic
)

var campaignPoints = map[Campaign]int{Low: 10, Standard: 15, High: 20, Epic: 25}

// ErrCampaign is shared verbatim with the other implementations.
var ErrCampaign = errors.New("purchase type must be one of low, standard, high, or epic.")

// Points is the purchase budget the campaign grants.
func (c Campaign) Points() int { return campaignPoints[c] }

// String is the campaign's name as it is written on the command line.
func (c Campaign) String() string {
	switch c {
	case Low:
		return "low"
	case High:
		return "high"
	case Epic:
		return "epic"
	default:
		return "standard"
	}
}

// ParseCampaign reads a campaign type by first letter, as the original did.
func ParseCampaign(s string) (Campaign, error) {
	switch upper := strings.ToUpper(strings.TrimSpace(s)); {
	case strings.HasPrefix(upper, "E"):
		return Epic, nil
	case strings.HasPrefix(upper, "H"):
		return High, nil
	case strings.HasPrefix(upper, "S"):
		return Standard, nil
	case strings.HasPrefix(upper, "L"):
		return Low, nil
	}
	return Standard, ErrCampaign
}

// Built on first use rather than at init, so the dieroller command pays nothing
// for a table only pathfinder-character reads.
var spreadsByCost = sync.OnceValue(func() map[int][][]int {
	byCost := make(map[int][][]int)
	for spread := range WithRepetition(LegalScores(), Abilities) {
		cost := TotalCost(spread)
		byCost[cost] = append(byCost[cost], spread)
	}
	return byCost
})

// SpreadsByCost returns every legal spread, grouped by total purchase cost.
func SpreadsByCost() map[int][][]int { return spreadsByCost() }

// SpreadsFor returns the spreads that cost exactly points.
//
// Exact equality matches the original: a 15-point character spends all 15.
func SpreadsFor(points int) [][]int { return spreadsByCost()[points] }

// SpreadCount is how many distinct legal spreads exist: C(17, 6) = 12376.
func SpreadCount() int {
	total := 0
	for _, spreads := range spreadsByCost() {
		total += len(spreads)
	}
	return total
}

// Generate picks a random spread costing exactly points, sorted descending.
func (g *Generator) Generate(points int) ([]int, error) {
	spreads := SpreadsFor(points)
	if len(spreads) == 0 {
		return nil, fmt.Errorf("no legal ability spread costs exactly %d points.", points)
	}
	return spreads[g.roller.IntN(len(spreads))], nil
}
