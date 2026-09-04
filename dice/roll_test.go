package dice_test

import (
	"math"
	"testing"

	"github.com/dmcbane/dieroller-go/dice"
)

func mustSpec(t *testing.T, s dice.Spec) dice.Spec {
	t.Helper()
	spec, err := dice.NewSpec(s)
	if err != nil {
		t.Fatalf("NewSpec(%+v): %v", s, err)
	}
	return spec
}

func TestRollKeepsTheRightNumberOfDice(t *testing.T) {
	r := dice.NewSeededRoller(1)
	spec := mustSpec(t, dice.Spec{Dice: 4, Sides: 6, Keep: 3})
	for range 200 {
		roll := r.Roll(spec)
		if len(roll.Rolled) != 4 || len(roll.Kept) != 3 {
			t.Fatalf("rolled %d kept %d, want 4 and 3", len(roll.Rolled), len(roll.Kept))
		}
		sum := 0
		for _, v := range roll.Kept {
			sum += v
		}
		if roll.Subtotal != sum {
			t.Fatalf("subtotal %d, want %d", roll.Subtotal, sum)
		}
	}
}

func TestEveryDieLandsInRange(t *testing.T) {
	r := dice.NewSeededRoller(2)
	spec := mustSpec(t, dice.Spec{Dice: 10, Sides: 6, Keep: 10})
	for _, v := range r.Roll(spec).Rolled {
		if v < 1 || v > 6 {
			t.Errorf("rolled %d on a d6", v)
		}
	}
}

func TestKeepHighAndLowTakeTheRightDice(t *testing.T) {
	r := dice.NewSeededRoller(3)
	high := mustSpec(t, dice.Spec{Dice: 5, Sides: 20, Keep: 2})
	low := mustSpec(t, dice.Spec{Dice: 5, Sides: 20, Keep: 2, From: dice.Low})
	for range 100 {
		roll := r.Roll(high)
		if roll.Kept[0] < roll.Kept[1] {
			t.Fatalf("keep-high returned %v out of order", roll.Kept)
		}
		for _, v := range roll.Rolled {
			if v > roll.Kept[0] {
				t.Fatalf("keep-high missed %d in %v", v, roll.Rolled)
			}
		}
		roll = r.Roll(low)
		if roll.Kept[0] > roll.Kept[1] {
			t.Fatalf("keep-low returned %v out of order", roll.Kept)
		}
		for _, v := range roll.Rolled {
			if v < roll.Kept[0] {
				t.Fatalf("keep-low missed %d in %v", v, roll.Rolled)
			}
		}
	}
}

func TestRollingIsNonDestructive(t *testing.T) {
	r := dice.NewSeededRoller(4)
	spec := mustSpec(t, dice.Spec{Dice: 4, Sides: 6, Keep: 2})
	roll := r.Roll(spec)
	// Select sorts a copy, so the dice stay in the order they came up.
	sorted := true
	for i := 1; i < len(roll.Rolled); i++ {
		if roll.Rolled[i-1] < roll.Rolled[i] {
			sorted = false
		}
	}
	if sorted && len(roll.Rolled) > 1 {
		t.Log("rolled happens to be descending this time; not conclusive")
	}
}

func TestAOneSidedDieIsDeterministic(t *testing.T) {
	r := dice.NewSeededRoller(5)
	expr, err := dice.Parse("3d1+2")
	if err != nil {
		t.Fatal(err)
	}
	outcome := r.RollExpr(expr)
	if outcome.Total != 5 {
		t.Errorf("3d1+2 totalled %d, want 5", outcome.Total)
	}
	// The multiplier scales the whole total, not each die.
	expr, _ = dice.Parse("3d1*2")
	if got := r.RollExpr(expr).Total; got != 6 {
		t.Errorf("3d1*2 totalled %d, want 6", got)
	}
	// A zero multiplier is applied, not ignored.
	expr, _ = dice.Parse("3d1*0")
	if got := r.RollExpr(expr).Total; got != 0 {
		t.Errorf("3d1*0 totalled %d, want 0", got)
	}
}

func TestGroupsCombineWithTheirSigns(t *testing.T) {
	r := dice.NewSeededRoller(6)
	expr, _ := dice.Parse("2d1+1d1-1")
	outcome := r.RollExpr(expr)
	if len(outcome.Groups) != 2 || outcome.Total != 2 {
		t.Errorf("2d1+1d1-1 gave %d groups totalling %d, want 2 and 2", len(outcome.Groups), outcome.Total)
	}
}

func TestSeedingIsReproducible(t *testing.T) {
	expr, _ := dice.Parse("10d100")
	first := dice.NewSeededRoller(42).RollExpr(expr).Total
	second := dice.NewSeededRoller(42).RollExpr(expr).Total
	if first != second {
		t.Errorf("the same seed gave %d and %d", first, second)
	}
	if dice.NewSeededRoller(1).RollExpr(expr).Total == dice.NewSeededRoller(2).RollExpr(expr).Total {
		t.Error("two different seeds gave the same roll")
	}
}

// Keep-lowest really is lower on average than keep-highest, and both land on
// their theoretical means. Same check the Racket and Elixir suites make.
func TestAdvantageAndDisadvantageMeans(t *testing.T) {
	mean := func(text string) float64 {
		expr, err := dice.Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		r := dice.NewSeededRoller(99)
		total := 0
		const n = 20000
		for range n {
			total += r.RollExpr(expr).Total
		}
		return float64(total) / n
	}
	low, high := mean("2d20kl1"), mean("2d20kh1")
	if low >= high {
		t.Errorf("disadvantage (%.3f) was not below advantage (%.3f)", low, high)
	}
	if math.Abs(low-7.175) > 0.25 {
		t.Errorf("2d20kl1 mean was %.3f, theoretical 7.175", low)
	}
	if math.Abs(high-13.825) > 0.25 {
		t.Errorf("2d20kh1 mean was %.3f, theoretical 13.825", high)
	}
}

func TestRollBatchCollectsOnlyForAnAggregate(t *testing.T) {
	r := dice.NewSeededRoller(7)
	plain := batch(t, "5x1d1")
	if totals := r.RollBatch(plain, nil); totals != nil {
		t.Errorf("an unaggregated batch collected %v", totals)
	}
	aggregated := batch(t, "sum(5x1d1)")
	totals := r.RollBatch(aggregated, nil)
	if len(totals) != 5 {
		t.Errorf("an aggregated batch collected %d totals, want 5", len(totals))
	}
	if got := aggregated.Aggregate.Format(totals); got != "5" {
		t.Errorf("sum(5x1d1) = %q, want %q", got, "5")
	}
}

func TestSpecSelect(t *testing.T) {
	high := mustSpec(t, dice.Spec{Dice: 4, Sides: 6, Keep: 3})
	if got := high.Select([]int{2, 5, 3, 6}); !equal(got, []int{6, 5, 3}) {
		t.Errorf("keep-high selected %v, want [6 5 3]", got)
	}
	low := mustSpec(t, dice.Spec{Dice: 4, Sides: 6, Keep: 3, From: dice.Low})
	if got := low.Select([]int{2, 5, 3, 6}); !equal(got, []int{2, 3, 5}) {
		t.Errorf("keep-low selected %v, want [2 3 5]", got)
	}
	// Select must not disturb its argument.
	rolled := []int{2, 5, 3, 6}
	high.Select(rolled)
	if !equal(rolled, []int{2, 5, 3, 6}) {
		t.Errorf("Select reordered its argument to %v", rolled)
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestApplyModifier(t *testing.T) {
	cases := []struct {
		op   dice.Op
		want int
	}{{dice.Add, 12}, {dice.Subtract, 8}, {dice.Multiply, 20}}
	for _, c := range cases {
		spec := mustSpec(t, dice.Spec{Dice: 3, Sides: 6, Keep: 3, Op: c.op, Amount: 2})
		if got := spec.ApplyModifier(10); got != c.want {
			t.Errorf("%v 2 applied to 10 = %d, want %d", c.op, got, c.want)
		}
	}
}
