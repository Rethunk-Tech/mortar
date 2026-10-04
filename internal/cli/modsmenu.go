package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/gmcm"
)

func (c *cmd) modsMenu() error {
	a, err := c.need(2, "a game", "a profile", "a mod")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3]}
	if c.setFlag {
		if len(a) < 4 {
			return usageError{"mods menu --set needs page/index=value"}
		}
		if _, _, _, err := gmcm.ParseSetFlag(a[3]); err != nil {
			return usageError{err.Error()}
		}
		p.Value = a[3]
		var edits []gmcm.Edit
		if err := c.ask("mods.menu", p, &edits, readTimeout); err != nil {
			return err
		}
		return c.emit(edits, func() {
			for _, e := range edits {
				fmt.Fprintf(c.out, "%s/%d %s %v\n", e.Page, e.Index, e.Name, e.Value)
			}
		})
	}
	var lines []string
	if err := c.ask("mods.menu", p, &lines, readTimeout); err != nil {
		return err
	}
	return c.emit(lines, func() {
		for _, line := range lines {
			fmt.Fprintln(c.out, line)
		}
	})
}
