package dice

import "strconv"

// Batch is a whole roll as it was written: an expression, how many times to
// roll it, and what to do with the results.
//
// 4d6k3 is a batch of one, 6x4d6k3 a batch of six reported one by one, and
// sum(6x4d6k3) the same six reduced to a single number by an [Aggregate].
//
// This is what [ParseRoll] returns, and the only place the repeat count lives:
// an [Expr] describes one roll and knows nothing about being rolled again.
type Batch struct {
	Expr      Expr
	Repeat    int
	Aggregate Aggregate
}

// Notation renders the batch in canonical notation.
//
// Without an aggregate this is the expression alone, because each line of
// output describes one roll and the repeat count is not part of that roll. An
// aggregate makes the repeat part of the answer -- a sum of six is not a
// property of any one of them -- so it is rendered too.
func (b Batch) Notation() string {
	if b.Aggregate == NoAggregate {
		return b.Expr.Notation()
	}
	return b.Aggregate.Notation() + "(" + strconv.Itoa(b.Repeat) + "x" + b.Expr.Notation() + ")"
}
