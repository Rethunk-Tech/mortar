package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) modsPreset() error {
	a, err := c.need(2, "a game", "a profile", "a mod", "list, save, apply, or delete")
	if err != nil {
		return err
	}
	switch a[3] {
	case "list", "save", "apply", "delete":
	default:
		return usageError{"mods preset needs list, save, apply, or delete"}
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3], Sub: a[3]}
	if len(a) >= 5 {
		p.Name = a[4]
	} else if a[3] != "list" {
		return usageError{"mods preset " + a[3] + " needs a name"}
	}
	return show(c, "mods.preset", p, func(names []string) {
		for _, name := range names {
			fmt.Fprintln(c.out, name)
		}
	})
}
