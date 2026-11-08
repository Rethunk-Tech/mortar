package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func (c *cmd) modsPreset() error {
	a, err := c.need(2, "a game", "a profile", "a mod", "list, save, apply, or delete")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], IDs: a[2:3], Sub: a[3]}
	if len(a) >= 5 {
		p.Name = a[4]
	}
	return show(c, "mods.preset", p, func(names []string) {
		for _, name := range names {
			fmt.Fprintln(c.out, name)
		}
	})
}
