package main

import (
	"fmt"
	"regexp"
)

const longHelp = `Generates random Pathfinder ability scores by one of five methods.

Characters are listed weakest first, so the best roll is the last line on
screen.

Examples:

  pathfinder-character --classic -v --number 4
  pathfinder-character -s -n 3
  pathfinder-character --pool 3:3:4:6:4:4 -v
  pathfinder-character -p epic -n 2 --json`

var unknownFlagRx = regexp.MustCompile(`^unknown (?:shorthand )?flag: -+(\S+)`)

// pflag's own wording differs from the other implementations'; this brings an
// unknown option back in line with them.
func reworded(err error) error {
	if m := unknownFlagRx.FindStringSubmatch(err.Error()); m != nil {
		dashes := "--"
		if len(m[1]) == 1 {
			dashes = "-"
		}
		return fmt.Errorf("unrecognized option %s%s.", dashes, m[1])
	}
	return err
}
