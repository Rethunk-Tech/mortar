package cli

import (
	"fmt"
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

func (c *cmd) sweep() error {
	a, err := c.need(1, "a game")
	if err != nil {
		return err
	}
	return show(c, "sweep", control.Params{Game: a[0]}, func(r launchsvc.SweepReport) { printSweep(c, r) })
}

func printSweep(c *cmd, r launchsvc.SweepReport) {
	if !r.Triggered {
		fmt.Fprintln(c.out, "No game or SMAPI version change.")
		return
	}
	if !r.NeedsAttention() {
		fmt.Fprintf(c.out, "%s %s / SMAPI %s: nothing to fix.\n", r.GameName, r.GameVersion, r.SMAPIVersion)
		return
	}
	rows := make([][]string, 0)
	for _, p := range r.Profiles {
		if len(p.Broken) == 0 && p.MissingDeps == 0 {
			continue
		}
		if p.MissingDeps > 0 {
			rows = append(rows, []string{p.Name, "", "missing deps", strconv.Itoa(p.MissingDeps)})
		}
		for _, m := range p.Broken {
			rows = append(rows, []string{p.Name, m.Name, m.Status, sweepFixLabel(m.Fix)})
		}
	}
	c.table("PROFILE\tMOD\tSTATUS\tFIX", rows)
}

func sweepFixLabel(fix string) string {
	switch fix {
	case "update":
		return "Update"
	case "off":
		return "Turn off"
	default:
		return "No fix yet"
	}
}
