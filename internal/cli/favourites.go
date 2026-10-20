package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

var marked = map[string]string{"freeze": "frozen", "unfreeze": "unfrozen", "favourite": "starred", "unfavourite": "unstarred"}

// profileMark freezes, unfreezes, stars or unstars a profile.
func (c *cmd) profileMark(sub string) error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "profile."+sub, control.Params{Game: a[0], Profile: a[1]}, func(p profile.Profile) {
		fmt.Fprintf(c.out, "%s: %s.\n", p.Name, marked[sub])
	})
}

func (c *cmd) modsFavourite(sub string) error {
	if sub == "favourites" {
		a, err := c.need(2, "a game")
		if err != nil {
			return err
		}
		return show(c, "mods.favourites", control.Params{Game: a[0]}, func(keys []string) {
			for _, k := range keys {
				fmt.Fprintln(c.out, k)
			}
		})
	}
	a, err := c.need(2, "a game", "a profile", "one or more mod ids")
	if err != nil {
		return err
	}
	return show(c, "mods."+sub, control.Params{Game: a[0], Profile: a[1], IDs: a[2:]}, func(keys []string) {
		fmt.Fprintf(c.out, "%d starred mods.\n", len(keys))
	})
}
