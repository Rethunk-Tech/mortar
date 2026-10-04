package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
)

func (c *cmd) savesCheck() error {
	a, err := c.need(2, "a game", "a save")
	if err != nil {
		return err
	}
	profileID := ""
	if len(a) > 2 {
		profileID = a[2]
	}
	return show(c, "saves.check", control.Params{Game: a[0], Name: a[1], Profile: profileID}, func(ch savessvc.SaveCheck) {
		if len(ch.Missing) == 0 {
			fmt.Fprintf(c.out, "%s has every recorded mod.\n", ch.Farm)
			return
		}
		names := make([]string, 0, len(ch.Missing))
		for _, m := range ch.Missing {
			names = append(names, m.Name)
		}
		fmt.Fprintf(c.out, "This save was last played with %d mods this profile lacks (%s).\n", len(ch.Missing), strings.Join(names, ", "))
		if ch.LastProfileExists {
			fmt.Fprintf(c.out, "Last profile: %s\n", ch.LastProfileID)
		}
	})
}
