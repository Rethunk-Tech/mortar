package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) modsFiles() error {
	a, err := c.need(2, "a game", "a profile", "a mod")
	if err != nil {
		return err
	}
	var rows []control.ModExtraFile
	if err := c.ask("mods.files", control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3]}, &rows, readTimeout); err != nil {
		return err
	}
	if c.json {
		keys := make([]string, 0, len(rows))
		for _, row := range rows {
			keys = append(keys, row.Key)
		}
		return c.emit(keys, func() {})
	}
	return c.emit(rows, func() {
		if len(rows) == 0 {
			fmt.Fprintln(c.out, "This mod has no linked extra files.")
			return
		}
		t := make([][]string, 0, len(rows))
		for _, row := range rows {
			t = append(t, []string{row.Key, row.Label})
		}
		c.table("KEY\tLABEL", t)
	})
}
