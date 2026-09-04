package dice

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Dice notation.
//
//	roll       := aggregate '(' repeated ')'
//	            | aggregate ':' repeated
//	            | repeated
//	repeated   := (integer 'x')? expression
//	aggregate  := 'sum' | 'avg' | 'high' | 'low' | 'median'   -- or an alias
//	expression := term (('+' | '-') term)* ('*' integer)?
//	term       := dice | integer
//	dice       := integer 'd' integer selector?
//	selector   := 'k' ('h' | 'l')? integer     -- keep, defaulting to highest
//	            | 'd' ('h' | 'l')  integer     -- drop
//
// so 4d6k3, 2d20kl1 (roll with disadvantage), 4d6dl1 (drop the lowest) and
// 2d6+1d8-1 are all accepted. Drop always needs its direction letter, because a
// bare d is already the dice separator.
//
// A leading repeat count rolls the same expression several times, so 6x4d6k3
// rolls four dice keeping the best three, six times over. The count is not part
// of the expression itself and does not appear in the rendered notation, since
// that describes a single roll.
//
// An aggregate wraps the whole thing and reduces those repeats to one number:
// sum(6x4d6k3), avg(100x1d20). The parenthesised form is the one to reach for,
// but a shell will eat unquoted parentheses, so sum:6x4d6k3 means exactly the
// same thing and needs no quoting.
//
// Dropping is stored as keeping from the opposite end, so 4d6dl1 and 4d6k3
// produce the same spec and both render as 4D6K3.
// Messages shared verbatim with the Elixir and Racket implementations; see the
// note on the errors in spec.go.
var (
	ErrRepeat       = errors.New("repeat count must be greater than 0.")
	ErrNoExpression = errors.New("no dice expression given.")
)

var (
	tokenRx  = regexp.MustCompile(`^([-+*])?(\d+[dD]\d+(?:[kK][hHlL]?\d+|[dD][hHlL]\d+)?|\d+)`)
	diceRx   = regexp.MustCompile(`^(\d+)[dD](\d+)(?:([kK])([hHlL]?)(\d+)|([dD])([hHlL])(\d+))?$`)
	repeatRx = regexp.MustCompile(`^(\d+)[xX](.+)$`)
	aggRx    = regexp.MustCompile(`^([a-zA-Z]+)(?:\((.*)\)|:(.+))$`)
	spaceRx  = regexp.MustCompile(`\s+`)
)

// ParseRoll parses a whole roll: an optional aggregate, an optional <n>x repeat
// count, then an expression.
//
// Unlike [Parse] this insists the expression contain at least one dice group,
// so a bare constant is reported rather than silently rolling nothing.
func ParseRoll(s string) (Batch, error) {
	original := strings.TrimSpace(s)

	kind, rest, err := splitAggregate(strip(original))
	if err != nil {
		return Batch{}, err
	}

	repeat := 1
	if m := repeatRx.FindStringSubmatch(rest); m != nil {
		repeat, _ = strconv.Atoi(m[1])
		rest = m[2]
		if repeat < 1 {
			return Batch{}, ErrRepeat
		}
	}

	expr, err := parseReportedAs(rest, original)
	if err != nil {
		return Batch{}, err
	}
	if len(expr.Specs()) == 0 {
		return Batch{}, fmt.Errorf("expression contains no dice: %q", original)
	}
	return Batch{Expr: expr, Repeat: repeat, Aggregate: kind}, nil
}

// An aggregate wraps the whole roll rather than sitting inside the expression,
// so it is peeled off before anything else is parsed. The : spelling exists
// because a shell would eat unquoted parentheses.
func splitAggregate(text string) (Aggregate, string, error) {
	m := aggRx.FindStringSubmatch(text)
	if m == nil {
		return NoAggregate, text, nil
	}
	kind, ok := ParseAggregate(m[1])
	if !ok {
		return NoAggregate, "", unknownAggregate(m[1])
	}
	rest := m[2]
	if m[3] != "" {
		rest = m[3]
	}
	return kind, rest, nil
}

func unknownAggregate(name string) error {
	names := AggregateNames()
	return fmt.Errorf("unknown aggregate %q; use %s, or %s.",
		name, strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
}

// Parse parses a dice expression.
func Parse(s string) (Expr, error) {
	return parseReportedAs(s, strings.TrimSpace(s))
}

// By the time ParseRoll gets here it has stripped the whitespace and peeled off
// the aggregate and the repeat count, none of which the roller wants to see
// quoted back at them: an error names the roll as it was written.
func parseReportedAs(s, original string) (Expr, error) {
	tokens, err := scan(strip(s), original)
	if err != nil {
		return Expr{}, err
	}
	return build(tokens, original)
}

func strip(s string) string { return spaceRx.ReplaceAllString(s, "") }

type token struct{ sign, body string }

func scan(s, original string) ([]token, error) {
	if s == "" {
		return nil, ErrNoExpression
	}
	var tokens []token
	for s != "" {
		m := tokenRx.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("could not parse dice notation: %q", original)
		}
		sign := m[1]
		if sign == "" {
			sign = "+"
		}
		tokens = append(tokens, token{sign, m[2]})
		s = s[len(m[0]):]
	}
	return tokens, nil
}

func build(tokens []token, original string) (Expr, error) {
	var expr Expr
	for _, t := range tokens {
		// A scale must be the final token, so anything following one is an error.
		if expr.HasScale {
			return Expr{}, fmt.Errorf("a * multiplier must come last: %q", original)
		}
		if t.sign == "*" {
			amount, err := strconv.Atoi(t.body)
			if err != nil {
				return Expr{}, fmt.Errorf("a * multiplier must be a whole number: %q", original)
			}
			expr.Scale, expr.HasScale = amount, true
			continue
		}
		sign := Plus
		if t.sign == "-" {
			sign = Minus
		}
		term, err := parseTerm(sign, t.body)
		if err != nil {
			return Expr{}, err
		}
		expr.Terms = append(expr.Terms, term)
	}
	return expr, nil
}

func parseTerm(sign Sign, body string) (Term, error) {
	m := diceRx.FindStringSubmatch(body)
	if m == nil {
		value, err := strconv.Atoi(body)
		if err != nil {
			return Term{}, fmt.Errorf("could not parse dice notation: %q", body)
		}
		return ConstantTerm(sign, value), nil
	}

	count, _ := strconv.Atoi(m[1])
	sides, _ := strconv.Atoi(m[2])
	keep, from := selection(count, m)

	spec, err := NewSpec(Spec{Dice: count, Sides: sides, Keep: keep, From: from})
	if err != nil {
		return Term{}, err
	}
	return DiceTerm(sign, spec), nil
}

// The capture groups are: 3 and 4 and 5 for a keep selector, 6 and 7 and 8 for
// a drop selector, and all empty when there is none.
func selection(count int, m []string) (keep int, from From) {
	switch {
	case m[5] != "": // keep, defaulting to the high end
		keep, _ = strconv.Atoi(m[5])
		return keep, direction(m[4], High)
	case m[8] != "": // drop is keeping from the opposite end: drop the lowest 1
		dropped, _ := strconv.Atoi(m[8])                       // of 4d6 is keep the highest 3, and
		return count - dropped, opposite(direction(m[7], Low)) // drop the highest 1 is keep the lowest 3
	default: // no selector: every die counts
		return count, High
	}
}

func direction(letter string, fallback From) From {
	switch strings.ToLower(letter) {
	case "l":
		return Low
	case "h":
		return High
	}
	return fallback
}

func opposite(f From) From {
	if f == Low {
		return High
	}
	return Low
}

// ParseModifier parses a standalone modifier argument. A leading +, -, or *
// selects the operation; without one, + is assumed.
func ParseModifier(s string) (Op, int, error) {
	trimmed := strings.TrimSpace(s)
	op := Add
	if trimmed != "" {
		switch trimmed[0] {
		case '+':
			op, trimmed = Add, trimmed[1:]
		case '-':
			op, trimmed = Subtract, trimmed[1:]
		case '*':
			op, trimmed = Multiply, trimmed[1:]
		}
	}
	amount, err := strconv.Atoi(strings.TrimSpace(trimmed))
	if err != nil {
		return Add, 0, fmt.Errorf("could not parse modifier: %q", s)
	}
	return op, amount, nil
}
