package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/modreport"
)

func (c *cmd) modsReport() error {
	a, err := c.need(3, "a game", "a profile", "a mod id (SMAPI id)")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], IDs: a[2:3], Run: c.run}
	return show(c, "mods.report", p, func(res modreport.Result) {
		fmt.Fprint(c.out, res.Text)
		if res.URL != "" {
			fmt.Fprintln(c.out)
			fmt.Fprintln(c.out, res.URL)
		}
	})
}
