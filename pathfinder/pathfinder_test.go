package pathfinder_test

import (
	"testing"

	"github.com/dmcbane/dieroller-go/dice"
	"github.com/dmcbane/dieroller-go/pathfinder"
)

func TestAbilityTableBoundaries(t *testing.T) {
	costs := map[int]int{1: -25, 10: 0, 18: 17, 45: 307}
	for score, want := range costs {
		if got, ok := pathfinder.Cost(score); !ok || got != want {
			t.Errorf("Cost(%d) = %d, want %d", score, got, want)
		}
	}
	bonuses := map[int]int{1: -5, 10: 0, 18: 4, 45: 17}
	for score, want := range bonuses {
		if got, ok := pathfinder.Bonus(score); !ok || got != want {
			t.Errorf("Bonus(%d) = %d, want %d", score, got, want)
		}
	}
	for _, out := range []int{0, 46, -1} {
		if _, ok := pathfinder.Cost(out); ok {
			t.Errorf("Cost(%d) reported ok", out)
		}
	}
}

// Every bonus follows floor((score - 10) / 2).
func TestBonusFollowsThePublishedProgression(t *testing.T) {
	for score := 1; score <= 45; score++ {
		want := (score - 10) / 2
		if (score-10)%2 != 0 && score < 10 {
			want--
		}
		got, _ := pathfinder.Bonus(score)
		if got != want {
			t.Errorf("Bonus(%d) = %d, want %d", score, got, want)
		}
	}
}

func TestCostRisesMonotonically(t *testing.T) {
	for score := 2; score <= 45; score++ {
		this, _ := pathfinder.Cost(score)
		prev, _ := pathfinder.Cost(score - 1)
		if this <= prev {
			t.Errorf("Cost(%d) = %d is not above Cost(%d) = %d", score, this, score-1, prev)
		}
	}
}

func TestStandardArrayCostsTheStandardBudget(t *testing.T) {
	array := []int{15, 14, 13, 12, 10, 8}
	if got := pathfinder.TotalCost(array); got != 15 {
		t.Errorf("the standard array costs %d, want 15", got)
	}
	if got := pathfinder.TotalBonus(array); got != 5 {
		t.Errorf("the standard array bonus is %d, want 5", got)
	}
}

func TestCombinationsCount(t *testing.T) {
	for n := 1; n <= 6; n++ {
		for k := 0; k <= 3; k++ {
			pool := make([]int, n)
			for i := range pool {
				pool[i] = i + 1
			}
			seen := 0
			for range pathfinder.WithRepetition(pool, k) {
				seen++
			}
			if want := pathfinder.CountWithRepetition(n, k); seen != want {
				t.Errorf("n=%d k=%d yielded %d, want %d", n, k, seen, want)
			}
		}
	}
	// The ability score case yields C(17, 6).
	if got := pathfinder.CountWithRepetition(12, 6); got != 12376 {
		t.Errorf("C(17,6) = %d, want 12376", got)
	}
}

func TestCombinationsHaveNoDuplicatesAndPreserveOrder(t *testing.T) {
	seen := map[[4]int]bool{}
	for c := range pathfinder.WithRepetition([]int{1, 2, 3, 4, 5, 6}, 4) {
		key := [4]int{c[0], c[1], c[2], c[3]}
		if seen[key] {
			t.Fatalf("duplicate combination %v", c)
		}
		seen[key] = true
	}
	// A descending pool yields descending combinations.
	for c := range pathfinder.WithRepetition(pathfinder.LegalScores(), 3) {
		for i := 1; i < len(c); i++ {
			if c[i-1] < c[i] {
				t.Fatalf("combination %v is not descending", c)
			}
		}
	}
}

func TestPurchaseTable(t *testing.T) {
	if got := pathfinder.SpreadCount(); got != 12376 {
		t.Errorf("the purchase table holds %d spreads, want 12376", got)
	}
	// Each campaign has the spread count the Racket brute force produced.
	counts := map[int]int{10: 225, 15: 262, 20: 280, 25: 272}
	for points, want := range counts {
		if got := len(pathfinder.SpreadsFor(points)); got != want {
			t.Errorf("%d points has %d spreads, want %d", points, got, want)
		}
	}
	// Every spread is grouped under its true cost, uses only legal scores, and
	// is sorted descending.
	for cost, spreads := range pathfinder.SpreadsByCost() {
		for _, spread := range spreads {
			if pathfinder.TotalCost(spread) != cost {
				t.Fatalf("%v is filed under %d but costs %d", spread, cost, pathfinder.TotalCost(spread))
			}
			if len(spread) != 6 {
				t.Fatalf("%v does not have six scores", spread)
			}
			for i, score := range spread {
				if score < 7 || score > 18 {
					t.Fatalf("%v contains an illegal score", spread)
				}
				if i > 0 && spread[i-1] < score {
					t.Fatalf("%v is not descending", spread)
				}
			}
		}
	}
	if got := pathfinder.SpreadsFor(9999); len(got) != 0 {
		t.Errorf("an unreachable budget has %d spreads", len(got))
	}
}

func TestCampaigns(t *testing.T) {
	cases := map[string]pathfinder.Campaign{
		"low": pathfinder.Low, "standard": pathfinder.Standard,
		"high": pathfinder.High, "epic": pathfinder.Epic,
		// Parsed by first letter, as the original did.
		"l": pathfinder.Low, "Epic Fantasy": pathfinder.Epic, "S": pathfinder.Standard,
	}
	for in, want := range cases {
		got, err := pathfinder.ParseCampaign(in)
		if err != nil || got != want {
			t.Errorf("ParseCampaign(%q) = %v %v, want %v", in, got, err, want)
		}
	}
	points := map[pathfinder.Campaign]int{
		pathfinder.Low: 10, pathfinder.Standard: 15, pathfinder.High: 20, pathfinder.Epic: 25,
	}
	for campaign, want := range points {
		if got := campaign.Points(); got != want {
			t.Errorf("%v grants %d points, want %d", campaign, got, want)
		}
	}
	if _, err := pathfinder.ParseCampaign("banana"); err == nil {
		t.Error("an unknown purchase type was accepted")
	} else if err.Error() != "purchase type must be one of low, standard, high, or epic." {
		t.Errorf("unexpected message %q", err)
	}
}

func TestPoolValidation(t *testing.T) {
	cases := map[string]string{
		"3/3/3/3/3":       "dice per attribute must specify die quantity for six attributes.",
		"2/3/3/3/3/10":    "a minimum of 3 dice must be used for each attribute.",
		"3/3/3/3/3/3":     "you must specify a total of twenty-four dice for the pool.",
		"3/3/3/3/3/three": "dice per attribute must be numbers separated by , / or :.",
	}
	for in, want := range cases {
		_, err := pathfinder.ParsePool(in)
		if err == nil || err.Error() != want {
			t.Errorf("ParsePool(%q) error = %v, want %q", in, err, want)
		}
	}
	// Commas, slashes, and colons all separate.
	for _, in := range []string{"3:3:4:6:4:4", "3,3,4,6,4,4", "3/3/4/6/4/4"} {
		counts, err := pathfinder.ParsePool(in)
		if err != nil {
			t.Fatalf("ParsePool(%q): %v", in, err)
		}
		if len(counts) != 6 || counts[3] != 6 {
			t.Errorf("ParsePool(%q) = %v", in, counts)
		}
	}
	if got := pathfinder.DefaultPool(); len(got) != 6 || got[0] != 4 {
		t.Errorf("DefaultPool() = %v, want four dice per ability", got)
	}
}

func TestAbilitySpecs(t *testing.T) {
	cases := map[pathfinder.MethodKind]string{
		pathfinder.MethodClassic:  "3D6",
		pathfinder.MethodStandard: "4D6K3",
		pathfinder.MethodHeroic:   "2D6+6",
	}
	for kind, want := range cases {
		specs, err := pathfinder.AbilitySpecs(pathfinder.Method{Kind: kind})
		if err != nil {
			t.Fatalf("AbilitySpecs(%v): %v", kind, err)
		}
		if len(specs) != 6 {
			t.Fatalf("AbilitySpecs(%v) gave %d specs, want 6", kind, len(specs))
		}
		if got := specs[0].Notation(); got != want {
			t.Errorf("AbilitySpecs(%v)[0] = %q, want %q", kind, got, want)
		}
	}
}

func TestGeneratedCharacters(t *testing.T) {
	g := pathfinder.NewGenerator(dice.NewSeededRoller(11))
	for _, m := range []pathfinder.Method{
		{Kind: pathfinder.MethodClassic},
		{Kind: pathfinder.MethodStandard},
		{Kind: pathfinder.MethodHeroic},
		{Kind: pathfinder.MethodPool, Pool: []int{3, 3, 4, 6, 4, 4}},
		{Kind: pathfinder.MethodPurchase, Points: 25},
	} {
		characters, err := g.Characters(m, 20)
		if err != nil {
			t.Fatalf("Characters(%v): %v", m, err)
		}
		if len(characters) != 20 {
			t.Fatalf("got %d characters, want 20", len(characters))
		}
		// Sorted weakest first, so the best roll is the last line on screen.
		for i := 1; i < len(characters); i++ {
			if characters[i-1].BonusTotal > characters[i].BonusTotal {
				t.Fatalf("%v is not sorted by bonus", m)
			}
		}
		for _, c := range characters {
			if len(c.Abilities) != 6 {
				t.Fatalf("a character has %d abilities", len(c.Abilities))
			}
			if c.BonusTotal != pathfinder.TotalBonus(c.Abilities) {
				t.Fatal("bonus_total disagrees with the abilities")
			}
		}
	}
	if _, err := g.Characters(pathfinder.Method{Kind: pathfinder.MethodStandard}, 0); err == nil {
		t.Error("zero characters was accepted")
	}
}

// Purchase characters spend exactly the campaign budget, and are not always in
// descending order, which made STR the best stat every time.
func TestPurchaseCharacters(t *testing.T) {
	g := pathfinder.NewGenerator(dice.NewSeededRoller(12))
	for _, points := range []int{10, 15, 20, 25} {
		characters, err := g.Characters(pathfinder.Method{Kind: pathfinder.MethodPurchase, Points: points}, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range characters {
			if c.CostTotal != points {
				t.Fatalf("a %d-point character cost %d", points, c.CostTotal)
			}
		}
	}

	characters, _ := g.Characters(pathfinder.Method{Kind: pathfinder.MethodPurchase, Points: 15}, 200)
	descending := 0
	for _, c := range characters {
		sorted := true
		for i := 1; i < len(c.Abilities); i++ {
			if c.Abilities[i-1] < c.Abilities[i] {
				sorted = false
			}
		}
		if sorted {
			descending++
		}
	}
	if descending > 40 {
		t.Errorf("%d of 200 purchase characters were in descending order", descending)
	}
}
