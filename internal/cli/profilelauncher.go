package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

func (c *cmd) profileShortcut() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "profile.shortcut", control.Params{Game: a[0], Profile: a[1], Remove: c.removeFlag}, func(res control.ShortcutResult) {
		if res.Removed {
			fmt.Fprintln(c.out, "Removed the shortcut for that profile.")
			return
		}
		fmt.Fprintf(c.out, "Wrote a shortcut at %s.\n", res.Path)
	})
}

func (c *cmd) profileSteam() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	if c.removeFlag {
		return usageError{"profile steam does not support --remove; remove the entry from Steam"}
	}
	return show(c, "profile.steam", control.Params{Game: a[0], Profile: a[1]}, func(res control.SteamShortcutResult) {
		switch res.Result {
		case steam.Unchanged:
			fmt.Fprintln(c.out, "That profile is already in Steam.")
		case steam.Updated:
			fmt.Fprintln(c.out, "Updated that profile's entry in Steam; the change shows the next time Steam starts.")
		case steam.Added:
			fmt.Fprintln(c.out, "Added that profile to Steam; it shows in your library the next time Steam starts.")
		}
	})
}
