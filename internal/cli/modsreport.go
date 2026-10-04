package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/modreport"
)

func (c *cmd) modsReport() error {
	a, err := c.need(3, "a game", "a profile", "a mod id (SMAPI UniqueID)")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3], Run: c.run}
	var res modreport.Result
	if err := c.ask("mods.report", p, &res, readTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprint(c.out, res.Text)
		if res.URL != "" {
			fmt.Fprintln(c.out)
			fmt.Fprintln(c.out, res.URL)
		}
	})
}
