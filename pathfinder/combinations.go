package pathfinder

import "iter"

// Combinations with repetition (multisets).
//
// The Racket original built its ability-score tables with six nested loops and
// deduplicated the results into a set, doing 12^6 = 2,985,984 iterations to
// arrive at 12,376 distinct sets. Enumerating the multisets directly produces
// exactly those 12,376 with no duplicates to discard.

// WithRepetition yields every k-element multiset drawn from pool.
//
// Each result preserves the ordering of pool, so a descending pool yields
// descending combinations. Every yielded slice is freshly allocated, so a
// caller may keep it.
func WithRepetition[T any](pool []T, k int) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		if k == 0 {
			yield(nil)
			return
		}
		if len(pool) == 0 {
			return
		}

		// Walks non-decreasing index tuples like an odometer: O(k) work per
		// element and constant memory, so taking the first few of an enormous
		// space is cheap.
		indices := make([]int, k)
		highest := len(pool) - 1
		for {
			combination := make([]T, k)
			for i, idx := range indices {
				combination[i] = pool[idx]
			}
			if !yield(combination) {
				return
			}

			// Advance the rightmost index that is not yet saturated, then reset
			// every index to its right to that same value.
			i := k - 1
			for i >= 0 && indices[i] == highest {
				i--
			}
			if i < 0 {
				return
			}
			next := indices[i] + 1
			for ; i < k; i++ {
				indices[i] = next
			}
		}
	}
}

// CountWithRepetition is how many k-element multisets a pool of n items yields:
// C(n + k - 1, k).
func CountWithRepetition(n, k int) int { return binomial(n+k-1, k) }

func binomial(n, k int) int {
	if k > n {
		return 0
	}
	result := 1
	for i := 1; i <= k; i++ {
		result = result * (n - k + i) / i
	}
	return result
}
