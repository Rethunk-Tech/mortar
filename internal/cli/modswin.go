package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) modsWin() error {
	a, err := c.need(2, "a game", "a profile", "a winner and a loser")
	if err != nil {
		return err
	}
	if len(a) < 4 {
		return usageError{"mods win needs a winner and a loser"}
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:4], Remove: c.undoFlag}
	return show(c, "mods.win", p, func(got profile.Profile) {
		if p.Remove {
			fmt.Fprintf(c.out, "Undid %s winning over %s.\n", a[2], a[3])
			return
		}
		fmt.Fprintf(c.out, "Made %s win over %s.\n", a[2], a[3])
	})
}
