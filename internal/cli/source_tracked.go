package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func (c *cmd) sourceTracked() error {
	if !c.missing {
		return usageError{"source tracked needs --missing"}
	}
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "source.trackedMissing", control.Params{Game: a[0], Profile: a[1], Source: c.sourceFlag}, func(mods []nexus.TrackedMod) {
		for _, im := range mods {
			fmt.Fprintf(c.out, "%d\t%s\n", im.ModID, im.DomainName)
		}
	})
}
