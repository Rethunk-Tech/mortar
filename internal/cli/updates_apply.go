package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (c *cmd) updatesApply() error {
	if !c.everywhere {
		return usageError{"updates apply needs --everywhere"}
	}
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], UniqueIDs: a[1:]}
	var r profile.EverywhereResult
	if err := c.call("updates.apply", p, &r, installTimeout); err != nil {
		return err
	}
	return c.emit(r, func() {
		if len(r.Updated) == 0 && len(r.Skipped) == 0 {
			fmt.Fprintln(c.out, "No profiles to update.")
			return
		}
		rows := make([][]string, 0, len(r.Updated)+len(r.Skipped))
		for _, h := range r.Updated {
			rows = append(rows, []string{h.Name, h.OldKey, "updated"})
		}
		for _, s := range r.Skipped {
			rows = append(rows, []string{s.Name, "", s.Reason})
		}
		c.table("PROFILE\tFROM\tSTATUS", rows)
	})
}
