package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) modsConfig() error {
	a, err := c.need(2, "a game", "a profile", "a mod")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:3]}
	switch {
	case len(a) >= 5:
		p.Key, p.Value = a[3], a[4]
	case len(a) == 4:
		return usageError{"mods config needs a field and a value"}
	}
	var fields []profile.ConfigField
	if err := c.ask("mods.config", p, &fields, readTimeout); err != nil {
		return err
	}
	return c.emit(fields, func() {
		for _, f := range fields {
			line := f.Path + " = " + f.Value
			if len(f.AllowValues) > 0 {
				line += "  [" + strings.Join(f.AllowValues, ", ") + "]"
			}
			fmt.Fprintln(c.out, line)
		}
	})
}
