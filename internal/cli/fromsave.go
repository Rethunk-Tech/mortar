package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
)

func (c *cmd) profileFromSave() error {
	a, err := c.need(2, "a game", "a save")
	if err != nil {
		return err
	}
	var got savessvc.FromSaveResult
	if err := c.ask("profile.fromSave", control.Params{Game: a[0], Name: a[1]}, &got, installTimeout); err != nil {
		return err
	}
	return c.emit(got, func() {
		fmt.Fprintf(c.out, "%s\t%s\n", got.Profile.ID, got.Profile.Name)
		if len(got.Added) > 0 {
			fmt.Fprintf(c.out, "Added from store: %d\n", len(got.Added))
		}
		if len(got.Queued) > 0 {
			fmt.Fprintf(c.out, "Queued: %d\n", len(got.Queued))
		}
		if len(got.Missing) > 0 {
			fmt.Fprintf(c.out, "Could not add: %d\n", len(got.Missing))
		}
	})
}
