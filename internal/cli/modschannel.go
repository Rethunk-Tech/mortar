package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func (c *cmd) modsChannel() error {
	a, err := c.need(2, "a game", "a profile", "a mod", "main, optional, or beta")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3], Value: a[3]}
	return show(c, "mods.channel", p, func(rows []control.ModRow) {
		fmt.Fprintf(c.out, "Update channel for %s is %s.\n", a[2], a[3])
	})
}
