// Package dice rolls dice described in dice notation.
//
// The package is pure with respect to IO: every entry point returns data, and
// formatting lives in the commands under cmd/. The only state is the random
// source, which a [Roller] owns so that a run can be made reproducible.
package dice

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// From says which end of the sorted roll the kept dice come from.
type From int

const (
	High From = iota
	Low
)

// Op is the modifier applied to the sum of the kept dice.
type Op int

const (
	Add Op = iota
	Subtract
	Multiply
)

// String renders an operator as it appears in notation.
func (o Op) String() string {
	switch o {
	case Add:
		return "+"
	case Subtract:
		return "-"
	case Multiply:
		return "*"
	}
	return ""
}

// Validation messages are shared verbatim with the Elixir and Racket
// implementations, which is why they are capitalised and end in a period
// against the usual Go convention: they are user-facing text, not wrapped
// context. The order the checks run in matters too, so an input with more than
// one problem reports the same message in all three.
var (
	ErrDice      = errors.New("dice must be greater than 0.")
	ErrKeep      = errors.New("keep must be greater than 0.")
	ErrDiceKeep  = errors.New("dice must be greater than or equal to keep.")
	ErrSides     = errors.New("sides must be greater than 0.")
	ErrDirection = errors.New("keep direction must be high or low.")
)

// Spec is a validated dice-rolling request: roll Dice dice of Sides sides
// each, keep Keep of them taken From the high or the low end, then apply Op
// with Amount to the sum of what was kept.
//
// Dropping is expressed as keeping: dropping the lowest 1 of 4d6 is keeping
// the highest 3, and dropping the highest 1 is keeping the lowest 3.
//
// The modifier applies to the sum, not to each die, so Multiply scales the
// whole kept total.
type Spec struct {
	Dice   int
	Sides  int
	Keep   int
	From   From
	Op     Op
	Amount int
}

// NewSpec normalises and validates a spec.
//
// Keep is required rather than defaulting from a zero value: 4d6dl4 drops every
// die and must be rejected, and a Keep of 0 that quietly meant "all of them"
// would turn that into a legal roll. Callers that want every die say so.
func NewSpec(s Spec) (Spec, error) {
	s = s.normalized()
	if err := s.validate(); err != nil {
		return Spec{}, err
	}
	return s, nil
}

// Keeping every die is the same from either end, so that case is canonicalised
// to High to keep the rendered notation stable.
func (s Spec) normalized() Spec {
	if s.Keep == s.Dice {
		s.From = High
	}
	return s
}

func (s Spec) validate() error {
	switch {
	case s.Dice < 1:
		return ErrDice
	case s.Keep < 1:
		return ErrKeep
	case s.Dice < s.Keep:
		return ErrDiceKeep
	case s.Sides < 1:
		return ErrSides
	case s.From != High && s.From != Low:
		return ErrDirection
	}
	return nil
}

// Notation renders the spec in canonical dice notation.
//
// The keep clause is omitted when every die is kept, rendered K<n> when keeping
// from the high end and KL<n> from the low end. The modifier is omitted only
// when it is exactly +0: *0 and -0 still print, because they change the result.
func (s Spec) Notation() string {
	return strconv.Itoa(s.Dice) + "D" + strconv.Itoa(s.Sides) + s.keepPart() + s.modifierPart()
}

func (s Spec) keepPart() string {
	switch {
	case s.Keep == s.Dice:
		return ""
	case s.From == Low:
		return "KL" + strconv.Itoa(s.Keep)
	default:
		return "K" + strconv.Itoa(s.Keep)
	}
}

func (s Spec) modifierPart() string {
	if s.Op == Add && s.Amount == 0 {
		return ""
	}
	return fmt.Sprintf("%v%d", s.Op, s.Amount)
}

// Select sorts rolled dice into selection order for this spec and takes what is
// kept: highest first from the high end, lowest first from the low end. The
// argument is not modified.
func (s Spec) Select(rolled []int) []int {
	ordered := make([]int, len(rolled))
	copy(ordered, rolled)
	if s.From == Low {
		sort.Ints(ordered)
	} else {
		sort.Sort(sort.Reverse(sort.IntSlice(ordered)))
	}
	return ordered[:s.Keep]
}

// ApplyModifier applies the spec's modifier to an already-summed roll.
func (s Spec) ApplyModifier(sum int) int {
	switch s.Op {
	case Subtract:
		return sum - s.Amount
	case Multiply:
		return sum * s.Amount
	default:
		return sum + s.Amount
	}
}
