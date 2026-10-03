package cli

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/problems"
)

func (c *cmd) modsCompat() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	var rows []problems.Compat
	if err := c.ask("compatibility", control.Params{Game: a[0], Profile: a[1]}, &rows, readTimeout); err != nil {
		return err
	}
	var shown []problems.Compat
	for _, row := range rows {
		if row.Status == "" || row.Status == "ok" {
			continue
		}
		shown = append(shown, row)
	}
	return c.emit(shown, func() {
		if len(shown) == 0 {
			fmt.Fprintln(c.out, "No compatibility issues.")
			return
		}
		t := make([][]string, 0, len(shown))
		for _, row := range shown {
			t = append(t, []string{row.Name, row.Status, row.Summary})
		}
		c.table("NAME\tSTATUS\tSUMMARY", t)
	})
}
