# dieroller-go

[![CI](https://github.com/dmcbane/dieroller-go/actions/workflows/ci.yml/badge.svg)](https://github.com/dmcbane/dieroller-go/actions/workflows/ci.yml)

A Go implementation of [dieroller](https://github.com/dmcbane/dieroller-rkt): a command line die
roller and a Pathfinder character generator for tabletop RPG players.

Three commands are built from this repository:

- **`dieroller`** — roll dice written in dice notation, repeat them, and optionally reduce those
  repeats to their sum, average, median, or extreme.
- **`pathfinder-character`** — generate Pathfinder ability scores by the classic, standard,
  heroic, pool, or purchase method.
- **`ability-scores`** — write the ability score cost/bonus tables as CSV.

## Layout

All the logic lives in two packages that perform no IO, so they can be tested directly; the
commands are thin shells over them.

```
dice/           dice specs, notation, aggregates, rolling
pathfinder/     ability tables, combinatorics, purchase spreads, generation
cmd/
  dieroller/
  pathfinder-character/
  ability-scores/
```

The only dependency is [cobra](https://github.com/spf13/cobra) for argument parsing; everything
else is the standard library.

## dieroller

A roll is written entirely in dice notation, as a single argument.

```
dieroller [flags] <roll>

Flags:
  -v, --verbose     Show the notation and the dice that were kept.
  -j, --json        Emit one JSON object per roll instead of text.
      --seed int    Seed the random number generator for reproducible rolls.
  -V, --version     Show the version
  -h, --help        help for dieroller
```

### Dice notation

```
roll       := aggregate '(' repeated ')'
            | aggregate ':' repeated
            | repeated
repeated   := (integer 'x')? expression
aggregate  := 'sum' | 'avg' | 'high' | 'low' | 'median'   -- or an alias
expression := term (('+' | '-') term)* ('*' integer)?
term       := dice | integer
dice       := integer 'd' integer selector?
selector   := 'k' ('h' | 'l')? integer     -- keep, defaulting to highest
            | 'd' ('h' | 'l')  integer     -- drop
```

| notation | meaning |
|---|---|
| `1d20` | one twenty-sided die |
| `5d20` | five of them |
| `3d6+3` | three six-sided dice, plus three |
| `4d6k3`, `4d6kh3` | keep the highest three of four |
| `2d20kh1` | advantage |
| `2d20kl1` | disadvantage |
| `4d6dl1` | drop the lowest (the same roll as `4d6k3`) |
| `4d6dh1` | drop the highest |
| `2d6+1d8-1` | several groups and constants |
| `3d6*2` | double the total of the kept dice |
| `6x4d6k3` | roll the same expression six times |
| `sum(6x4d6k3)` | add those six rolls together |
| `sum:6x4d6k3` | the same, with nothing for a shell to eat |
| `avg:100x1d20` | the average of a hundred rolls |
| `max:2x1d20` | the better of two rolls |

A modifier applies to the sum of the kept dice, not to each die, so `3d6*2` doubles the total
rather than rolling `3d12`. Drop always needs its direction letter, since a bare `d` already
separates dice from sides; dropping is stored as keeping from the other end, so `4d6dl1` and
`4d6k3` are the same spec and both display as `4D6K3`.

The repeat count is not part of the expression and does not appear in the rendered notation,
which describes a single roll.

### Aggregating repeated rolls

`6x4d6k3` reports six rolls. An aggregate wraps the whole thing and reports one number instead:

| aggregate | aliases | what it reports |
|---|---|---|
| `sum` | `total` | every roll added together |
| `avg` | `average`, `mean` | their average, rounded to two places |
| `high` | `highest`, `max` | the best of them |
| `low` | `lowest`, `min` | the worst of them |
| `median` | `med` | the middle one |

Most shells treat unquoted parentheses as syntax of their own — fish reads `(...)` as command
substitution, bash as a subshell — so either quote the whole roll or use the colon form, which
parses identically.

Unlike the repeat count, an aggregate *does* appear in the rendered notation, and it takes the
repeat with it: a sum of six rolls is a property of all six, not of any one of them, so it
renders as `SUM(6x4D6K3)`. Aliases canonicalise the way `kh` does, so `max:2x1d20` renders as
`HIGH(2x1D20)`.

An average is rounded to two places in whole hundredths rather than by scaling a float. Forty
rolls totalling three average exactly 0.075, which rounds to `0.08`; the nearest `float64` to
0.075 is a hair below it, so rounding a float would report `0.07`. The Elixir and Racket ports
round exactly for the same reason, so all three agree on every value.

An aggregate needs every roll before it can report anything. Under `--verbose` the rolls still
appear as they are made and the summary follows them; an unaggregated repeat count streams and
collects nothing, so `dieroller 100000000x1d6 | head` costs no memory.

### Examples

```console
$ dieroller 4d6k3
13

$ dieroller 6x4d6k3 --verbose
4D6K3 (5 4 4) => 13
4D6K3 (5 2 1) => 8
4D6K3 (5 3 1) => 9
4D6K3 (3 2 2) => 7
4D6K3 (6 5 4) => 15
4D6K3 (4 3 2) => 9

$ dieroller 2d6+1d8-1 -v
2D6+1D8-1 (5 2) (6) => 12

$ dieroller "2d6 + 1d8"          # quote it if you write spaces
14

$ dieroller "sum(6x4d6k3)" -v
4D6K3 (6 4 4) => 14
4D6K3 (6 4 2) => 12
4D6K3 (1 1 1) => 3
4D6K3 (6 6 4) => 16
4D6K3 (5 3 2) => 10
4D6K3 (5 4 3) => 12
SUM(6x4D6K3) => 67

$ dieroller 3d6+2 --json
{"notation":"3D6+2","groups":[{"notation":"3D6","rolled":[4,1,3],"kept":[4,3,1],"sum":8}],"subtotal":10,"total":10}

$ dieroller avg:6x4d6k3 --json
{"notation":"AVG(6x4D6K3)","aggregate":"avg","expression":"4D6K3","repeat":6,"rolls":[14,12,3,16,10,12],"value":11.166666666666666}
```

An aggregated `--json` roll is one object for the batch rather than one per roll, carrying each
roll's total under `rolls` and the exact, unrounded result under `value`; only the text output
rounds.

### Migrating from the old arguments

The flags that used to describe the dice (`--dice`, `--sides`, `--keep`, `--modifier`,
`--iterations`) and the `<dice> <sides> <modifier> <keep>` positional form are gone; the notation
says all of it. The removed forms report their notation equivalent rather than failing blankly:

```console
$ dieroller 3 6 +6 2
the <dice> <sides> <modifier> <keep> arguments have been replaced by dice notation; try: dieroller 3d6k2+6

$ dieroller 2d6 + 1d8
a roll is one argument; quote the whole expression, for example: dieroller "2d6 + 1d8"
```

## pathfinder-character

```
pathfinder-character [flags]

Flags:
  -c, --classic          The classic method: 3D6 per ability.
  -s, --standard         The standard method: 4D6 keep high 3 per ability. (this is the default)
  -r, --heroic           The heroic method: 2D6 plus 6 per ability.
  -l, --pool string      The pool method: 24D6 split across the six abilities, minimum 3 each, as 3/3/3/3/3/9.
  -p, --purchase string  The purchase method: low, standard, high, or epic, granting 10, 15, 20, or 25 points.
  -v, --verbose          Display additional information.
  -n, --number int       Number of characters to roll. Must be greater than 0. (default 1)
  -j, --json             Emit one JSON object per character instead of text.
      --seed int         Seed the random number generator for reproducible characters.
  -V, --version          Show the version
```

Characters are listed weakest first, so the best roll is the last line on screen.

```console
$ pathfinder-character --classic -v --number 4
STR: 11 DEX: 7 CON: 9 INT: 8 WIS: 7 CHR: 8 (bonus -7, cost -12)
STR: 13 DEX: 12 CON: 7 INT: 7 WIS: 16 CHR: 9 (bonus 0, cost 6)
STR: 14 DEX: 5 CON: 13 INT: 8 WIS: 13 CHR: 13 (bonus 1, cost 3)
STR: 13 DEX: 15 CON: 10 INT: 12 WIS: 8 CHR: 13 (bonus 4, cost 13)

$ pathfinder-character -s -n 3
12 12 5 10 12 15
12 17 13 15 12 8
16 6 17 14 11 12

$ pathfinder-character --pool 3:3:4:6:4:4 -v
STR: 11 DEX: 12 CON: 11 INT: 18 WIS: 10 CHR: 10 (bonus 5, cost 21)
```

The purchase table — every legal spread of six scores drawn from 7..18, grouped by total cost —
is enumerated on first use as combinations with repetition, C(17, 6) = 12,376 spreads. The Racket
original arrived at the same set through a memoized 12⁶ = 2,985,984 nested-loop brute force.

## ability-scores

```console
$ ability-scores --out ./csv [--legal-only]
```

Writes `legal_scores.csv` (12,376 spreads of scores 7..18) and `uniq_scores.csv` (15,890,700
spreads of scores 1..45), each row carrying total purchase cost, total ability bonus, and the six
scores. The output is byte-identical to the Elixir and Racket implementations'.

## Building

Requires Go 1.24 or newer.

```console
$ go build -o bin/dieroller ./cmd/dieroller
$ go build -o bin/pathfinder-character ./cmd/pathfinder-character
$ go build -o bin/ability-scores ./cmd/ability-scores
```

Or install them onto your `PATH`:

```console
$ go install github.com/dmcbane/dieroller-go/cmd/...@latest
```

Tagged releases carry prebuilt Linux x86-64 executables and a `SHA256SUMS` file, attached by CI
from the same build it smoke tested.

## Testing

```console
$ go test ./...
```

The suite covers both packages with unit tests and both CLIs end to end through `--seed`.

One test reproduces the Racket original's 12⁶ = 2,985,984 nested-loop brute force and asserts it
yields exactly the same set of purchase spreads this implementation enumerates. It runs by
default and is skipped by `go test -short ./...`.

## Relationship to the other implementations

There are four implementations of these programs, kept deliberately in step:

| | Racket | Elixir | Go |
|---|---|---|---|
| repository | [dieroller-rkt](https://github.com/dmcbane/dieroller-rkt) | [dieroller-elixir](https://github.com/dmcbane/dieroller-elixir) | this one |
| dice notation, aggregates | yes | yes | yes |
| `--json`, `--seed` | no | yes | yes |
| CLI parsing | `racket/cmdline` | `OptionParser` | cobra |

### Verified against them

- **Notation strings** — identical for every argument form, including the `+0` suppression, the
  drop-to-keep canonicalisation, and the `SUM(6x4D6K3)` rendering of an aggregate.
- **Validation messages** — every message and both migration hints match byte for byte. Forty-nine
  deterministic invocations (one-sided dice and every error path) were run against all three
  implementations and diffed.
- **Aggregate values** — identical for every kind. Racket reduces in exact rationals; this port
  and the Elixir one round in whole hundredths to reach the same answers without a float in the
  way.
- **Purchase spreads** — the enumerated table reduces to SHA-256
  `66d08d1a…9211bcee`, the digest of the table dumped from Racket 8.16 itself, which the Elixir
  port also pins.
- **CSV output** — `ability-scores --legal-only` produces a `legal_scores.csv` byte-identical to
  the Elixir and Racket versions'.

`--seed` reproduces a run only within one implementation: the three seed different generators, so
the same seed does not give the same dice across languages.

## What changed from the original Go version

This repository previously tracked the pre-notation design and had drifted:

- The module path was `dieroller` while the code imported `github.com/dmcbane/dierollergo`, so
  the two commands could not both build from a clean checkout.
- `go.mod` required `kingpin.v2`, which nothing imported; argument parsing was hand-rolled over
  the standard `flag` package.
- `rand.Seed` was called once per constructor **and again after every single die**, which is both
  deprecated since Go 1.20 and a way to make a sequence less random rather than more.
- `pathfinder-character` fanned its characters out to goroutines and printed from inside them, so
  output order was nondeterministic and the "weakest first" ordering the other implementations
  guarantee was absent.
- There was no notation parser, no keep/drop selectors, no repeat count, no aggregates, no
  `--json`, no `--seed`, no CI, and a two-line README.

## License

MIT. See [LICENSE](LICENSE).
