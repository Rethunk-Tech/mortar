package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
)

func (c *cmd) who() error {
	a, err := c.need(1, "a game", "a profile", "a query")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], Query: strings.Join(a[2:], " ")}
	return show(c, "who", p, func(page contentpatcher.WhoChangesPage) { printAssetTargets(c, page.Targets) })
}

func (c *cmd) conflictsMap() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], Query: c.filter}
	return show(c, "conflicts.map", p, func(page contentpatcher.AssetMapPage) { printAssetTargets(c, page.Targets) })
}

func printAssetTargets(c *cmd, targets []contentpatcher.AssetTarget) {
	if len(targets) == 0 {
		fmt.Fprintln(c.out, "No assets.")
		return
	}
	rows := [][]string{}
	for _, target := range targets {
		label := target.Target
		if target.Key != "" {
			label += " " + target.Key
		}
		mods := []string{}
		winner := target.Winner.Local()
		for _, m := range target.Mods {
			name := m.ModName
			if name == "" {
				name = m.ModID.Local()
			}
			if m.Winner {
				name += "*"
				winner = m.ModName
			}
			mods = append(mods, fmt.Sprintf("%s %s", name, m.Action))
		}
		if winner == "" {
			winner = "unclear"
		}
		rows = append(rows, []string{label, fmt.Sprint(len(target.Mods)), strings.Join(mods, ", "), winner})
	}
	c.table("TARGET\tMODS\tCHANGES\tWINNER", rows)
}
