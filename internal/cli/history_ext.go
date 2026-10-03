package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) historyCmd() error {
	if len(c.args) > 1 && c.args[1] == "diff" {
		return c.historyDiff()
	}
	if len(c.args) > 1 && c.args[1] == "revert" {
		return c.historyRevertItem()
	}
	return c.historyAll()
}

func (c *cmd) historyDiff() error {
	a, err := c.need(2, "a game", "a profile", "snapshot a", "snapshot b")
	if err != nil {
		return err
	}
	var diff profile.HistoryDiff
	if err := c.ask("history.diff", control.Params{Game: a[0], Profile: a[1], Name: a[2], Value: a[3]}, &diff, readTimeout); err != nil {
		return err
	}
	return c.emit(diff, func() { printHistoryDiff(c, diff) })
}

func (c *cmd) historyRevertItem() error {
	a, err := c.need(2, "a game", "a profile", "an event id")
	if err != nil {
		return err
	}
	if c.item == "" {
		return usageError{"history revert needs --item"}
	}
	var p profile.Profile
	if err := c.ask("history.revert", control.Params{Game: a[0], Profile: a[1], Name: a[2], Value: c.item}, &p, readTimeout); err != nil {
		return err
	}
	return c.emit(p, func() { fmt.Fprintf(c.out, "%s\t%s\n", p.ID, p.Name) })
}

func (c *cmd) profileChanges() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	var diff profile.HistoryDiff
	if err := c.ask("profile.changes", control.Params{Game: a[0], Profile: a[1]}, &diff, readTimeout); err != nil {
		return err
	}
	return c.emit(diff, func() { printHistoryDiff(c, diff) })
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
		var rows []profile.HistoryEvent
		if err := c.ask("profile.good", p, &rows, readTimeout); err != nil {
			return err
		}
		return c.emit(rows, func() {
			t := [][]string{}
			for _, row := range rows {
				t = append(t, []string{row.At.Local().Format("2006-01-02 15:04"), row.Label})
			}
			c.table("TIME\tSUMMARY", t)
		})
	}
	if c.restore {
		var prof profile.Profile
		if err := c.ask("profile.good", p, &prof, readTimeout); err != nil {
			return err
		}
		return c.emit(prof, func() { fmt.Fprintf(c.out, "%s\t%s\n", prof.ID, prof.Name) })
	}
	var ev profile.HistoryEvent
	if err := c.ask("profile.good", p, &ev, readTimeout); err != nil {
		return err
	}
	return c.emit(ev, func() { fmt.Fprintf(c.out, "%s\t%s\n", ev.ID, ev.Label) })
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
