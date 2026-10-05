package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
)

func (c *cmd) modsMenu() error {
	a, err := c.need(2, "a game", "a profile", "a mod")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], IDs: a[2:3]}
	if c.setFlag {
		if len(a) < 4 {
			return usageError{"mods menu --set needs page/index=value"}
		}
		if _, _, _, err := gmcm.ParseSetFlag(a[3]); err != nil {
			return usageError{err.Error()}
		}
		p.Value = a[3]
		return show(c, "mods.menu", p, func(edits []gmcm.Edit) {
			for _, e := range edits {
				fmt.Fprintf(c.out, "%s/%d %s %v\n", e.Page, e.Index, e.Name, e.Value)
			}
		})
	}
	return show(c, "mods.menu", p, func(lines []string) {
		for _, line := range lines {
			fmt.Fprintln(c.out, line)
		}
	})
}
