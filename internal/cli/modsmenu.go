package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
)

func (c *cmd) modsMenu() error {
	a, err := c.need(2, "a game", "a profile", "a mod")
	if err != nil {
		return err
	}
	p := modTarget(a)
	if c.setFlag {
		spec := c.setSpec
		if !c.setEquals {
			if len(a) < 4 {
				return usageError{"mods menu --set needs page/index=value"}
			}
			spec = a[3]
		}
		if _, _, _, err := gmcm.ParseSetFlag(spec); err != nil {
			return usageError{err.Error()}
		}
		p.Value = spec
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
