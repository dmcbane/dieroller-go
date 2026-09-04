// Package pathfinder generates Pathfinder ability scores.
package pathfinder

// Ability score lookup tables.
//
// Scores 1..6 and 19..45 are not legal purchase values in Pathfinder; they are
// extrapolations of the published table, kept so that rolled characters (which
// can fall outside the purchase range) can still be priced and compared.
//
// Transcribed from the Racket source so all four implementations stay aligned
// and remain diffable against the original.
const (
	MinScore      = 1
	MaxScore      = 45
	MinLegalScore = 7
	MaxLegalScore = 18
	Abilities     = 6
)

// Indexed by score - 1.
var (
	costs = [MaxScore]int{
		-25, -20, -16, -12, -9, -6, -4, -2, -1, 0, // 1..10
		1, 2, 3, 5, 7, 10, 13, 17, 21, 26, // 11..20
		31, 37, 43, 50, 57, 65, 73, 82, 91, 101, // 21..30
		111, 122, 133, 145, 157, 170, 183, 197, 211, 226, // 31..40
		241, 257, 273, 290, 307, // 41..45
	}
	bonuses = [MaxScore]int{
		-5, -4, -4, -3, -3, -2, -2, -1, -1, 0, // 1..10
		0, 1, 1, 2, 2, 3, 3, 4, 4, 5, // 11..20
		5, 6, 6, 7, 7, 8, 8, 9, 9, 10, // 21..30
		10, 11, 11, 12, 12, 13, 13, 14, 14, 15, // 31..40
		15, 16, 16, 17, 17, // 41..45
	}
)

// Cost is the purchase cost of one ability score. Out-of-range scores report
// ok=false rather than panicking, since a rolled score is not bounds-checked
// before it gets here.
func Cost(score int) (int, bool) {
	if score < MinScore || score > MaxScore {
		return 0, false
	}
	return costs[score-1], true
}

// Bonus is the ability bonus of one ability score.
func Bonus(score int) (int, bool) {
	if score < MinScore || score > MaxScore {
		return 0, false
	}
	return bonuses[score-1], true
}

// TotalCost is the total purchase cost of a set of ability scores.
func TotalCost(scores []int) int {
	total := 0
	for _, s := range scores {
		c, _ := Cost(s)
		total += c
	}
	return total
}

// TotalBonus is the total ability bonus of a set of ability scores.
func TotalBonus(scores []int) int {
	total := 0
	for _, s := range scores {
		b, _ := Bonus(s)
		total += b
	}
	return total
}

// LegalScores returns the scores that are legal to buy with purchase points,
// highest first, which is the order the purchase table is built in.
func LegalScores() []int {
	scores := make([]int, 0, MaxLegalScore-MinLegalScore+1)
	for s := MaxLegalScore; s >= MinLegalScore; s-- {
		scores = append(scores, s)
	}
	return scores
}

// AllScores returns every score the tables cover, highest first.
func AllScores() []int {
	scores := make([]int, 0, MaxScore)
	for s := MaxScore; s >= MinScore; s-- {
		scores = append(scores, s)
	}
	return scores
}
