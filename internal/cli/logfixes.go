package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func (c *cmd) logFixes() error {
	p, err := c.runParams()
	if err != nil {
		return err
	}
	return show(c, "logs.fixes", p, func(found []launch.SMAPIProblem) {
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
