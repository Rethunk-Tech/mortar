package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) modsChannel() error {
	a, err := c.need(2, "a game", "a profile", "a mod", "main, optional, or beta")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3], Value: a[3]}
	var rows []control.ModRow
	if err := c.ask("mods.channel", p, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() {
		fmt.Fprintf(c.out, "Update channel for %s is %s.\n", a[2], a[3])
	})
}
