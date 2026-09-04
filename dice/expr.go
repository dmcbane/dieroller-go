package dice

import "strconv"

// Sign is the sign a term carries within an expression.
type Sign int

const (
	Plus Sign = iota
	Minus
)

// Term is one element of an expression: either a dice group or a constant.
// Exactly one of Spec and Constant is meaningful, as IsDice says.
type Term struct {
	Sign     Sign
	Spec     Spec
	Constant int
	IsDice   bool
}

// Expr is a whole dice expression: one or more signed terms, optionally scaled.
//
// A term is either a dice group (2d6, 4d6k3) or a plain constant, so 2d6+1d8-1
// is three terms. Scale carries a trailing *n, which multiplies the total of
// every term. HasScale distinguishes "no multiplier" from "*0", which is a
// real expression that always totals zero.
type Expr struct {
	Terms    []Term
	Scale    int
	HasScale bool
}

// DiceTerm builds a signed dice term.
func DiceTerm(sign Sign, spec Spec) Term {
	return Term{Sign: sign, Spec: spec, IsDice: true}
}

// ConstantTerm builds a signed constant term.
func ConstantTerm(sign Sign, value int) Term {
	return Term{Sign: sign, Constant: value}
}

// ExprFromSpec wraps a single spec as an expression, preserving its modifier.
func ExprFromSpec(s Spec) Expr {
	bare := s
	bare.Op, bare.Amount = Add, 0

	switch {
	case s.Op == Multiply:
		return Expr{Terms: []Term{DiceTerm(Plus, bare)}, Scale: s.Amount, HasScale: true}
	// An exactly-zero addition is the "no modifier" case and adds no term,
	// which is what keeps 3D6 rendering without a trailing +0.
	case s.Op == Add && s.Amount == 0:
		return Expr{Terms: []Term{DiceTerm(Plus, bare)}}
	case s.Op == Subtract:
		return Expr{Terms: []Term{DiceTerm(Plus, bare), ConstantTerm(Minus, s.Amount)}}
	default:
		return Expr{Terms: []Term{DiceTerm(Plus, bare), ConstantTerm(Plus, s.Amount)}}
	}
}

// Notation renders the expression in canonical notation.
func (e Expr) Notation() string {
	out := ""
	for i, t := range e.Terms {
		// A leading plus is implicit; every later term carries its sign.
		if i > 0 || t.Sign == Minus {
			if t.Sign == Minus {
				out += "-"
			} else {
				out += "+"
			}
		}
		if t.IsDice {
			out += t.Spec.Notation()
		} else {
			out += strconv.Itoa(t.Constant)
		}
	}
	if e.HasScale {
		out += "*" + strconv.Itoa(e.Scale)
	}
	return out
}

// Specs returns the dice groups in the expression, in order, ignoring
// constant terms.
func (e Expr) Specs() []Spec {
	var specs []Spec
	for _, t := range e.Terms {
		if t.IsDice {
			specs = append(specs, t.Spec)
		}
	}
	return specs
}
