package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) profileHealth() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "profile.health", control.Params{Game: a[0], Profile: a[1]}, func(rows []profile.HealthPoint) { c.printHealthTable(rows) })
}

func (c *cmd) printHealthTable(rows []profile.HealthPoint) {
	if len(rows) == 0 {
		fmt.Fprintln(c.out, "No health history yet.")
		return
	}
	table := [][]string{}
	for _, row := range rows {
		at := row.At.Local().Format("2006-01-02 15:04")
		table = append(table, []string{
			at,
			fmt.Sprintf("%d", row.Problems),
			fmt.Sprintf("%d", row.Warnings),
			fmt.Sprintf("%d", row.Updates),
		})
	}
	c.table("TIME\tPROBLEMS\tWARNINGS\tUPDATES", table)
}
