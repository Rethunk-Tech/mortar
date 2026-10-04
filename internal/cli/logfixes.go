package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func (c *cmd) logFixes() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	run := c.run
	if run == "" && len(c.args) > 4 {
		run = c.args[4]
	}
	return show(c, "logs.fixes", control.Params{Game: a[0], Profile: a[1], Run: run}, func(found []launch.SMAPIProblem) {
		if len(found) == 0 {
			fmt.Fprintln(c.out, "No recognised problems.")
			return
		}
		for _, p := range found {
			name := p.ModName
			if p.ModID != "" && p.ModID != p.ModName {
				name = p.ModName + " (" + p.ModID + ")"
			}
			extra := ""
			if p.Dependency != "" {
				extra = " needs " + p.Dependency
			}
			fmt.Fprintf(c.out, "%s  %s  %s%s\n  %s\n", p.Kind, p.Fix, name, extra, p.Detail)
		}
	})
}
