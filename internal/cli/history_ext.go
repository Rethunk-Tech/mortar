package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func (c *cmd) historyCmd() error {
	if len(c.args) > 1 && c.args[1] == "diff" {
		return c.historyDiff()
	}
	if len(c.args) > 1 && c.args[1] == "revert" {
		return c.historyRevertItem()
	}
	if len(c.args) > 1 && c.args[1] == "usage" {
		return c.historyUsage()
	}
	if len(c.args) > 1 && c.args[1] == "trim" {
		return c.historyTrim()
	}
	return c.historyAll()
}

func (c *cmd) historyDiff() error {
	a, err := c.need(2, "a game", "a profile", "snapshot a", "snapshot b")
	if err != nil {
		return err
	}
	return show(c, "history.diff", control.Params{Game: a[0], Profile: a[1], Name: a[2], Value: a[3]}, func(diff profile.HistoryDiff) { printHistoryDiff(c, diff) })
}

func (c *cmd) historyRevertItem() error {
	a, err := c.need(2, "a game", "a profile", "an event id")
	if err != nil {
		return err
	}
	if c.item == "" {
		return usageError{"history revert needs --item"}
	}
	return show(c, "history.revert", control.Params{Game: a[0], Profile: a[1], Name: a[2], Value: c.item}, func(p profile.Profile) { fmt.Fprintf(c.out, "%s\t%s\n", p.ID, p.Name) })
}

func (c *cmd) profileChanges() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "profile.changes", control.Params{Game: a[0], Profile: a[1]}, func(diff profile.HistoryDiff) { printHistoryDiff(c, diff) })
}

func (c *cmd) profileGood() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	if c.mark == c.restore && (c.mark || c.restore) {
		return usageError{"profile good needs --mark or --restore"}
	}
	p := control.Params{Game: a[0], Profile: a[1], All: c.mark, Force: c.restore}
	if !c.mark && !c.restore {
		return show(c, "profile.good", p, func(rows []profile.HistoryEvent) {
			t := [][]string{}
			for _, row := range rows {
				t = append(t, []string{row.At.Local().Format("2006-01-02 15:04"), row.Label})
			}
			c.table("TIME\tSUMMARY", t)
		})
	}
	if c.restore {
		return show(c, "profile.good", p, func(prof profile.Profile) { fmt.Fprintf(c.out, "%s\t%s\n", prof.ID, prof.Name) })
	}
	return show(c, "profile.good", p, func(ev profile.HistoryEvent) { fmt.Fprintf(c.out, "%s\t%s\n", ev.ID, ev.Label) })
}

func printHistoryDiff(c *cmd, diff profile.HistoryDiff) {
	if len(diff.Items) == 0 {
		fmt.Fprintln(c.out, "No differences.")
		return
	}
	rows := make([][]string, 0, len(diff.Items))
	for _, it := range diff.Items {
		rows = append(rows, []string{it.Kind, it.Name, it.Detail})
	}
	c.table("KIND\tMOD\tCHANGE", rows)
}

func playGroupsBlocking(groups []control.PlayIssueGroup) int {
	n := 0
	for _, g := range groups {
		if g.Kind != "changes" {
			n++
		}
	}
	return n
}
