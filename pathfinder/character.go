package pathfinder

// Abbreviations are the ability names, in roll order.
var Abbreviations = [Abilities]string{"STR", "DEX", "CON", "INT", "WIS", "CHR"}

// Character is a generated character: six ability scores plus their totals.
//
// Abilities are in STR, DEX, CON, INT, WIS, CHR order for the rolled methods.
// The purchase method has no roll order, so its spreads are shuffled before
// they get here.
type Character struct {
	Abilities  []int
	BonusTotal int
	CostTotal  int
}

// NewCharacter builds a character from six ability scores, computing both
// totals.
func NewCharacter(abilities []int) Character {
	return Character{
		Abilities:  abilities,
		BonusTotal: TotalBonus(abilities),
		CostTotal:  TotalCost(abilities),
	}
}

// Labeled pairs each ability with its abbreviation.
func (c Character) Labeled() []struct {
	Name  string
	Value int
} {
	out := make([]struct {
		Name  string
		Value int
	}, len(c.Abilities))
	for i, v := range c.Abilities {
		out[i].Name, out[i].Value = Abbreviations[i], v
	}
	return out
}
