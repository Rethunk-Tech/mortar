package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) profileShortcut() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	var res control.ShortcutResult
	if err := c.ask("profile.shortcut", control.Params{Game: a[0], Profile: a[1], Remove: c.removeFlag}, &res, readTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
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
	var res control.SteamShortcutResult
	if err := c.ask("profile.steam", control.Params{Game: a[0], Profile: a[1]}, &res, readTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		if res.Already {
			fmt.Fprintln(c.out, "That profile is already in Steam.")
			return
		}
		fmt.Fprintln(c.out, "Added that profile to Steam; it shows in your library the next time Steam starts.")
	})
}
