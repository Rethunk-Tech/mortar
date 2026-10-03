package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) nexusUntrack() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	if c.all == c.unused {
		return usageError{"nexus untrack needs exactly one of --all or --unused"}
	}
	var count int
	if err := c.ask("nexus.tracked", control.Params{Game: a[0]}, &count, readTimeout); err != nil {
		return err
	}
	if !c.yesFlag {
		if info, err := os.Stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return refusedError{"nexus untrack needs --yes when stdin is not a terminal"}
		}
		fmt.Fprintf(c.errOut, "Untrack %d mods? [y/N] ", count)
		var answer string
		if _, err := fmt.Fscan(os.Stdin, &answer); err != nil {
			return err
		}
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			return errors.New("cancelled")
		}
	}
	var result control.NexusUntrack
	if err := c.ask("nexus.untrack", control.Params{Game: a[0], Unused: c.unused}, &result, readTimeout); err != nil {
		return err
	}
	return c.emit(result, func() {
		fmt.Fprintf(c.out, "Untracked %d mods; %d remaining.\n", result.Untracked, result.Remaining)
		if result.StoppedForLimit {
			fmt.Fprintln(c.out, "Stopped at the Nexus API limit.")
		}
	})
}
