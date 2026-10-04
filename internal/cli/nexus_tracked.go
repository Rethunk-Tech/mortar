package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/nexus"
)

func (c *cmd) nexusTracked() error {
	if !c.missing {
		return usageError{"nexus tracked needs --missing"}
	}
	a, err := c.need(3, "a game", "a profile")
	if err != nil {
		return err
	}
	var mods []nexus.TrackedMod
	if err := c.ask("nexus.trackedMissing", control.Params{Game: a[0], Profile: a[1]}, &mods, readTimeout); err != nil {
		return err
	}
	return c.emit(mods, func() {
		for _, mod := range mods {
			fmt.Fprintf(c.out, "%d\t%s\n", mod.ModID, mod.DomainName)
		}
	})
}
