package cli

import (
	"fmt"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

const playTestTimeout = 4 * time.Minute

func (c *cmd) playTest() error {
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	var res launchsvc.TestLaunchResult
	if err := c.ask("play.test", control.Params{Game: a[0], Profile: a[1]}, &res, playTestTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		if res.ReachedTitle {
			fmt.Fprintln(c.out, "Reached the title screen")
			return
		}
		fmt.Fprintf(c.out, "Crashed: %s\n", res.Cause)
	})
}
