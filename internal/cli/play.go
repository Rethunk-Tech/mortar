package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
)

type playCheckError struct{ code int }

func (e playCheckError) Error() string { return "play check found issues" }

func (c *cmd) play() error {
	if !c.check {
		return usageError{"play needs --check"}
	}
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	var groups []control.PlayIssueGroup
	if err := c.ask("play.check", control.Params{Game: a[0], Profile: a[1]}, &groups, readTimeout); err != nil {
		return err
	}
	if err := c.emit(groups, func() { printPlayIssues(c, groups) }); err != nil {
		return err
	}
	if len(groups) > 0 {
		return playCheckError{code: 3}
	}
	return nil
}

func printPlayIssues(c *cmd, groups []control.PlayIssueGroup) {
	if len(groups) == 0 {
		fmt.Fprintln(c.out, "Ready to play.")
		return
	}
	for _, g := range groups {
		names := strings.Join(g.Names, ", ")
		if names == "" {
			fmt.Fprintf(c.out, "%s (%d)\n", g.Kind, g.Count)
			continue
		}
		fmt.Fprintf(c.out, "%s (%d): %s\n", g.Kind, g.Count, names)
	}
}
