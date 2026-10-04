package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func (c *cmd) gameSteamLaunchOption() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	if c.setFlag && c.clearFlag {
		return usageError{"use either --set or --clear, not both"}
	}
	return show(c, "game.steamLaunchOption", control.Params{Game: a[0], Set: c.setFlag, Clear: c.clearFlag}, func(res control.SteamLaunchOptionResult) {
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
