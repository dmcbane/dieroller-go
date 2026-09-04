package pathfinder

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dmcbane/dieroller-go/dice"
)

// Five generation methods are supported, named as they are on the command line.
type MethodKind int

const (
	MethodClassic  MethodKind = iota // 3D6 per ability
	MethodStandard                   // 4D6 keep the highest 3, per ability
	MethodHeroic                     // 2D6 plus 6, per ability
	MethodPool                       // 24D6 split across the six abilities
	MethodPurchase                   // a random spread costing exactly Points
)

// Method is a generation method and whatever parameter it carries.
type Method struct {
	Kind   MethodKind
	Pool   []int // MethodPool only
	Points int   // MethodPurchase only
}

// Name is how the method is reported in JSON output.
func (m Method) Name() string {
	switch m.Kind {
	case MethodClassic:
		return "classic"
	case MethodHeroic:
		return "heroic"
	case MethodPool:
		counts := make([]string, len(m.Pool))
		for i, c := range m.Pool {
			counts[i] = strconv.Itoa(c)
		}
		return "pool " + strings.Join(counts, "/")
	case MethodPurchase:
		return "purchase " + strconv.Itoa(m.Points)
	default:
		return "standard"
	}
}

const (
	poolTotal   = 24
	poolMinimum = 3
)

// Validation messages carried over verbatim from the Racket original.
var (
	ErrPoolSize    = errors.New("dice per attribute must specify die quantity for six attributes.")
	ErrPoolMinimum = errors.New("a minimum of 3 dice must be used for each attribute.")
	ErrPoolTotal   = errors.New("you must specify a total of twenty-four dice for the pool.")
	ErrPoolNumbers = errors.New("dice per attribute must be numbers separated by , / or :.")
	ErrCount       = errors.New("number of characters must be greater than 0.")
)

// Generator produces characters from one random source.
type Generator struct {
	roller *dice.Roller
}

// NewGenerator returns a generator drawing on the given roller.
func NewGenerator(roller *dice.Roller) *Generator { return &Generator{roller: roller} }

// Characters generates count characters, sorted ascending by total ability
// bonus.
//
// Sorting weakest-first matches the original, which put the best roll last so
// it is the line left on screen.
func (g *Generator) Characters(m Method, count int) ([]Character, error) {
	if count < 1 {
		return nil, ErrCount
	}
	characters := make([]Character, 0, count)
	for range count {
		abilities, err := g.abilities(m)
		if err != nil {
			return nil, err
		}
		characters = append(characters, NewCharacter(abilities))
	}
	sort.SliceStable(characters, func(i, j int) bool {
		return characters[i].BonusTotal < characters[j].BonusTotal
	})
	return characters, nil
}

// Spreads are stored sorted descending so they deduplicate as multisets. Handed
// to a character in that order, STR would always be the highest ability, so the
// assignment order is randomised here, at the boundary where a spread becomes a
// character rather than in the table itself.
func (g *Generator) abilities(m Method) ([]int, error) {
	if m.Kind == MethodPurchase {
		spread, err := g.Generate(m.Points)
		if err != nil {
			return nil, err
		}
		shuffled := make([]int, len(spread))
		copy(shuffled, spread)
		g.roller.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		return shuffled, nil
	}

	specs, err := AbilitySpecs(m)
	if err != nil {
		return nil, err
	}
	scores := make([]int, len(specs))
	for i, spec := range specs {
		scores[i] = g.roller.Roll(spec).Total
	}
	return scores, nil
}

// AbilitySpecs returns the per-ability dice specs a rolled method uses.
func AbilitySpecs(m Method) ([]dice.Spec, error) {
	switch m.Kind {
	case MethodClassic:
		return uniformSpecs(dice.Spec{Dice: 3, Sides: 6, Keep: 3})
	case MethodStandard:
		return uniformSpecs(dice.Spec{Dice: 4, Sides: 6, Keep: 3})
	case MethodHeroic:
		return uniformSpecs(dice.Spec{Dice: 2, Sides: 6, Keep: 2, Op: dice.Add, Amount: 6})
	case MethodPool:
		if err := ValidatePool(m.Pool); err != nil {
			return nil, err
		}
		specs := make([]dice.Spec, len(m.Pool))
		for i, count := range m.Pool {
			spec, err := dice.NewSpec(dice.Spec{Dice: count, Sides: 6, Keep: poolMinimum})
			if err != nil {
				return nil, err
			}
			specs[i] = spec
		}
		return specs, nil
	}
	return nil, fmt.Errorf("method has no dice specs")
}

func uniformSpecs(s dice.Spec) ([]dice.Spec, error) {
	spec, err := dice.NewSpec(s)
	if err != nil {
		return nil, err
	}
	specs := make([]dice.Spec, Abilities)
	for i := range specs {
		specs[i] = spec
	}
	return specs, nil
}

// ParsePool reads a pool distribution such as "3/3/3/3/3/9". Commas, slashes,
// and colons all separate, as in the original.
func ParsePool(s string) ([]int, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '/' || r == ':'
	})
	counts := make([]int, 0, len(fields))
	for _, field := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return nil, ErrPoolNumbers
		}
		counts = append(counts, n)
	}
	if err := ValidatePool(counts); err != nil {
		return nil, err
	}
	return counts, nil
}

// DefaultPool is four dice per ability.
func DefaultPool() []int {
	pool := make([]int, Abilities)
	for i := range pool {
		pool[i] = 4
	}
	return pool
}

// ValidatePool checks a pool distribution, reporting the same message and in
// the same order as the original.
func ValidatePool(counts []int) error {
	if len(counts) != Abilities {
		return ErrPoolSize
	}
	total := 0
	for _, c := range counts {
		if c < poolMinimum {
			return ErrPoolMinimum
		}
		total += c
	}
	if total != poolTotal {
		return ErrPoolTotal
	}
	return nil
}
