package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) modsByAuthor() error {
	a, err := c.need(3, "a game", "an author name")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Query: strings.Join(a[1:], " ")}
	var rows []profile.AuthorMod
	if err := c.ask("mods.by-author", p, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() {
		if len(rows) == 0 {
			fmt.Fprintln(c.out, "No mods for that author.")
			return
		}
		for _, mod := range rows {
			fmt.Fprintf(c.out, "%s (%s)\n", mod.Name, mod.UniqueID)
			for _, prof := range mod.Profiles {
				state := enabledLabel(prof.Enabled)
				fmt.Fprintf(c.out, "  %s · %s · %s\n", prof.ProfileName, prof.Version, state)
			}
		}
	})
}
