package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (c *cmd) profileSet() error {
	a, err := c.need(2, "a game", "a profile", "a key", "a value")
	if err != nil {
		return err
	}
	return show(c, "profile.set", control.Params{Game: a[0], Profile: a[1], Key: a[2], Value: strings.Join(a[3:], " ")}, func(p profile.Profile) {
		fmt.Fprintln(c.out, p.ID)
	})
}
