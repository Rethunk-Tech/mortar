package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/control"
)

// bisectCmd runs a crash check: start takes a game and profile and prints the job id, status and stop take the id.
func (c *cmd) bisectCmd() error {
	if len(c.args) < 2 {
		return usageError{"bisect needs start, status or stop"}
	}
	sub := c.args[1]
	var p control.Params
	switch sub {
	case "start":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		p = control.Params{Game: a[0], Profile: a[1]}
	case "status", "stop":
		a, err := c.need(2, "a crash check id")
		if err != nil {
			return err
		}
		p = control.Params{Name: a[0]}
	default:
		return usageError{"unknown bisect command " + sub}
	}
	return show(c, "bisect."+sub, p, func(st bisect.Status) {
		fmt.Fprintf(c.out, "%s  %s  step %d of %d, %d mods left\n", st.ID, st.State, st.Step, st.Total, st.ModsLeft)
		if st.Error != "" {
			fmt.Fprintln(c.out, st.Error)
		}
		if st.Result != nil {
			for _, m := range st.Result.Mods {
				fmt.Fprintf(c.out, "%+v\n", m)
			}
		}
	})
}
