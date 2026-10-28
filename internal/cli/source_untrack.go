package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func (c *cmd) sourceUntrack() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	if c.all == c.unused {
		return usageError{"source untrack needs exactly one of --all or --unused"}
	}
	var count int
	if err := c.call("source.tracked", control.Params{Game: a[0], Source: c.sourceFlag, Unused: c.unused}, &count, readTimeout); err != nil {
		return err
	}
	if err := c.confirm("source untrack", fmt.Sprintf("Untrack %d mods?", count)); err != nil {
		return err
	}
	return show(c, "source.untrack", control.Params{Game: a[0], Source: c.sourceFlag, Unused: c.unused}, func(result control.NexusUntrack) {
		fmt.Fprintf(c.out, "Untracked %d mods; %d remaining.\n", result.Untracked, result.Remaining)
		if result.StoppedForLimit {
			fmt.Fprintln(c.out, "Stopped at the Nexus API limit.")
		}
	})
}
