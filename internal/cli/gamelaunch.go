package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) gameCmd() error {
	if len(c.args) < 2 {
		return usageError{"game needs steam-launch-option"}
	}
	if c.args[1] != "steam-launch-option" {
		return usageError{"unknown game command " + c.args[1]}
	}
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	if c.setFlag && c.clearFlag {
		return usageError{"use either --set or --clear, not both"}
	}
	var res control.SteamLaunchOptionResult
	if err := c.ask("game.steamLaunchOption", control.Params{Game: a[0], Set: c.setFlag, Clear: c.clearFlag}, &res, readTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		switch {
		case res.Set:
			if res.Options == "" {
				fmt.Fprintln(c.out, "Set Steam launch options so the game starts through its loader.")
				return
			}
			fmt.Fprintf(c.out, "Set Steam launch options to %s.\n", res.Options)
		case res.Cleared:
			if res.Options == "" {
				fmt.Fprintln(c.out, "Removed the loader from Steam launch options.")
				return
			}
			fmt.Fprintf(c.out, "Removed the loader from Steam launch options; they are now %s.\n", res.Options)
		default:
			if res.Options == "" {
				fmt.Fprintln(c.out, "Steam has no launch options set for that game.")
				return
			}
			fmt.Fprintf(c.out, "Steam launch options are %s.\n", res.Options)
		}
	})
}
