package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/problems"
)

func (c *cmd) who() error {
	a, err := c.need(1, "a game", "a profile", "a query")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], Query: strings.Join(a[2:], " ")}
	var page problems.WhoChangesPage
	if err := c.ask("who", p, &page, readTimeout); err != nil {
		return err
	}
	return c.emit(page, func() { printAssetTargets(c, page.Targets) })
}

func (c *cmd) conflictsMap() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], Query: c.filter}
	var page problems.AssetMapPage
	if err := c.ask("conflicts.map", p, &page, readTimeout); err != nil {
		return err
	}
	return c.emit(page, func() { printAssetTargets(c, page.Targets) })
}

func printAssetTargets(c *cmd, targets []problems.AssetTarget) {
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
		winner := target.Winner
		for _, m := range target.Mods {
			name := m.ModName
			if name == "" {
				name = m.ModID
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
