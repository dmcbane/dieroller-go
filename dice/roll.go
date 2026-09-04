package dice

import (
	"math/rand/v2"
)

// Roll is the outcome of rolling a [Spec].
//
// Rolled holds every die in the order it came up; Kept holds the selected dice
// in selection order. Subtotal is the sum of Kept before the modifier, Total
// after it.
type Roll struct {
	Spec     Spec
	Rolled   []int
	Kept     []int
	Subtotal int
	Total    int
}

// Outcome is the result of rolling a whole [Expr].
//
// Groups holds one [Roll] per dice term, in the order they appear in the
// expression; constant terms contribute to the totals but have no group.
// Subtotal is the signed sum of every term, Total that sum after any *n.
type Outcome struct {
	Expr     Expr
	Groups   []Roll
	Subtotal int
	Total    int
}

// Roller rolls dice from a random source.
//
// Holding the source rather than reaching for the global one is what makes
// --seed work and what lets a test roll a known sequence.
type Roller struct {
	rng *rand.Rand
}

// NewRoller returns a roller seeded from the runtime's entropy.
func NewRoller() *Roller {
	return &Roller{rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
}

// NewSeededRoller returns a roller that produces the same sequence for the same
// seed. The sequence is this implementation's own: the Elixir and Racket ports
// seed different generators, so a seed reproduces a run only within one of them.
func NewSeededRoller(seed int64) *Roller {
	return &Roller{rng: rand.New(rand.NewPCG(uint64(seed), uint64(seed)+1))}
}

// IntN returns a uniform random integer in [0, n) from the roller's source, so
// a caller that needs randomness for something other than dice -- picking a
// purchase spread, shuffling abilities -- stays on the same seeded stream.
func (r *Roller) IntN(n int) int { return r.rng.IntN(n) }

// Shuffle permutes n elements using the roller's source.
func (r *Roller) Shuffle(n int, swap func(i, j int)) { r.rng.Shuffle(n, swap) }

// Roll rolls a spec once.
func (r *Roller) Roll(s Spec) Roll {
	rolled := make([]int, s.Dice)
	for i := range rolled {
		rolled[i] = r.rng.IntN(s.Sides) + 1
	}
	kept := s.Select(rolled)
	subtotal := 0
	for _, v := range kept {
		subtotal += v
	}
	return Roll{
		Spec:     s,
		Rolled:   rolled,
		Kept:     kept,
		Subtotal: subtotal,
		Total:    s.ApplyModifier(subtotal),
	}
}

// RollExpr rolls a whole expression, one group per dice term.
func (r *Roller) RollExpr(e Expr) Outcome {
	groups := make([]Roll, 0, len(e.Terms))
	subtotal := 0
	for _, t := range e.Terms {
		value := t.Constant
		if t.IsDice {
			roll := r.Roll(t.Spec)
			groups = append(groups, roll)
			value = roll.Total
		}
		if t.Sign == Minus {
			subtotal -= value
		} else {
			subtotal += value
		}
	}
	total := subtotal
	if e.HasScale {
		total = subtotal * e.Scale
	}
	return Outcome{Expr: e, Groups: groups, Subtotal: subtotal, Total: total}
}

// RollBatch rolls a batch, calling each with every outcome as it is made.
//
// The callback is what lets a large repeat count stream to the terminal instead
// of being collected first. Totals come back only for an aggregated batch,
// which is the only caller that needs every roll at once; an unaggregated
// 100000000x1d6 therefore costs no memory.
func (r *Roller) RollBatch(b Batch, each func(Outcome)) []int {
	var totals []int
	if b.Aggregate != NoAggregate {
		totals = make([]int, 0, b.Repeat)
	}
	for i := 0; i < b.Repeat; i++ {
		outcome := r.RollExpr(b.Expr)
		if each != nil {
			each(outcome)
		}
		if totals != nil {
			totals = append(totals, outcome.Total)
		}
	}
	return totals
}
