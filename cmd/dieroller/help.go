package main

const longHelp = `A roll is written entirely in dice notation, as a single argument:

  <roll>       := <repeated> | <aggregate>(<repeated>) | <aggregate>:<repeated>
  <repeated>   := [<repeat>x] <expression>
  <expression> := <term> (('+' | '-') <term>)* ['*' <integer>]
  <term>       := <dice> | <integer>
  <dice>       := <count>d<sides>[<selector>]

  <selector> is one of
    k<n>, kh<n>  keep the highest <n> dice (k defaults to highest)
    kl<n>        keep the lowest <n> dice   (roll with disadvantage)
    dl<n>        drop the lowest <n> dice
    dh<n>        drop the highest <n> dice

  <aggregate> reduces a repeated roll to a single number, and is one of
    sum, total          every roll added together
    avg, average, mean  their average, to two places
    high, highest, max  the best of them
    low, lowest, min    the worst of them
    median, med         the middle one

A modifier applies to the sum of the kept dice, not to each die, so 3d6*2
doubles the total rather than rolling 3d12. Quote the expression if you write
it with spaces. Most shells eat unquoted parentheses, so either quote the whole
roll or use the colon form, which means exactly the same thing.

Examples:

  dieroller 1d20                 a single twenty-sided die
  dieroller 5d20                 five of them
  dieroller 3d6+3                three six-sided dice, plus three
  dieroller 4d6k3                keep the best three of four
  dieroller 2d20kh1              advantage
  dieroller 2d20kl1              disadvantage
  dieroller 4d6dl1               drop the lowest of four
  dieroller 2d6+1d8-1            several dice groups and a constant
  dieroller 3d6*2                double the total
  dieroller 6x4d6k3              roll the same thing six times
  dieroller 6x4d6k3 --verbose    show the dice that were kept
  dieroller "2d6 + 1d8"          spaces are fine when quoted
  dieroller "sum(6x4d6k3)"       add those six rolls up
  dieroller sum:6x4d6k3          the same, with nothing for a shell to eat
  dieroller avg:100x1d20         the average of a hundred rolls
  dieroller max:2x1d20           the better of two rolls
  dieroller "sum(6x4d6k3)" -v    show each roll, then the sum`
