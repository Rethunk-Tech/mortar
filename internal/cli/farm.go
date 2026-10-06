package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
)

// profileFarm shares a Stardew profile's mods for multiplayer and compares them with a host's list: export prints the
// list, check names what differs, fix queues the downloads.
func (c *cmd) profileFarm() error {
	if len(c.args) < 3 {
		return usageError{"profile farm needs export, check or fix"}
	}
	verb := c.args[2]
	switch verb {
	case "export":
		a, err := c.need(3, "a game", "a profile")
		if err != nil {
			return err
		}
		var list packsvc.FarmList
		if err := c.call("pack.farmExport", control.Params{Game: a[0], Profile: a[1]}, &list, readTimeout); err != nil {
			return err
		}
		c.json = true
		return c.emit(list, nil)
	case "check", "fix":
		a, err := c.need(3, "a game", "a profile", "the host's list file or JSON")
		if err != nil {
			return err
		}
		text := strings.Join(a[2:], " ")
		if raw, err := os.ReadFile(a[2]); err == nil {
			text = string(raw)
		}
		p := control.Params{Game: a[0], Profile: a[1], Value: text}
		if verb == "fix" {
			var res packsvc.FarmFix
			if err := c.call("pack.farmFix", p, &res, installTimeout); err != nil {
				return err
			}
			return c.emit(res, func() {
				fmt.Fprintf(c.out, "queued %d downloads\n", res.Queued)
				if len(res.Manual) > 0 {
					fmt.Fprintf(c.out, "Install by hand (no download source): %s\n", strings.Join(res.Manual, ", "))
				}
			})
		}
		return show(c, "pack.farmCheck", p, func(r packsvc.FarmCheck) {
			if len(r.Rows) == 0 {
				fmt.Fprintf(c.out, "This profile matches %s.\n", r.Host)
				return
			}
			for _, row := range r.Rows {
				fmt.Fprintf(c.out, "%s\t%s\thost %s\tyours %s\n", row.State, row.Name, row.Host, row.Mine)
			}
		})
	default:
		return usageError{"unknown profile farm command " + verb}
	}
}
