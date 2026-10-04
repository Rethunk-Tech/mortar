package cli

import (
	"fmt"
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/dlwatch"
)

func (c *cmd) downloads() error {
	if len(c.args) > 1 {
		if c.args[1] != "install" {
			return usageError{"unknown downloads command " + c.args[1]}
		}
		if len(c.args) < 3 {
			return usageError{"downloads install needs a list number"}
		}
		n, err := strconv.Atoi(c.args[2])
		if err != nil || n < 1 {
			return usageError{"downloads install needs a list number"}
		}
		return c.installMethod("downloads.install", control.Params{Name: strconv.Itoa(n)})
	}
	var list []dlwatch.Item
	if err := c.ask("downloads", control.Params{}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		if len(list) == 0 {
			fmt.Fprintln(c.out, "No Downloads-folder archives this session.")
			return
		}
		t := [][]string{}
		for _, it := range list {
			t = append(t, []string{strconv.Itoa(it.N), it.Name, it.State, it.Path})
		}
		c.table("N\tNAME\tSTATE\tPATH", t)
	})
}
