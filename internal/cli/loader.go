package cli

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func (c *cmd) loader() error {
	if len(c.args) < 2 {
		return usageError{"loader needs versions, install or pin"}
	}
	switch c.args[1] {
	case "versions":
		a, err := c.need(2, "a game")
		if err != nil {
			return err
		}
		return show(c, "loader.versions", control.Params{Game: a[0], Loader: c.loaderFlag}, func(rows []string) {
			for _, v := range rows {
				fmt.Fprintln(c.out, v)
			}
		})
	case "install":
		a, err := c.need(2, "a game", "a version")
		if err != nil {
			return err
		}
		var st loader.Status
		if err := c.call("loader.install", control.Params{Game: a[0], Name: a[1], Loader: c.loaderFlag}, &st, installTimeout); err != nil {
			return err
		}
		return c.emit(st, func() {
			fmt.Fprintf(c.out, "Installed loader %s.\n", st.Version)
		})
	case "pin":
		a, err := c.need(2, "a game", "a version")
		if err != nil {
			return err
		}
		value := a[1]
		if value == "latest" {
			value = ""
		}
		if err := c.call("loader.pin", control.Params{Game: a[0], Value: value, Loader: c.loaderFlag}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]string{"game": a[0], "pin": value}, func() {
			if value == "" {
				fmt.Fprintln(c.out, "The loader follows the latest release.")
				return
			}
			fmt.Fprintf(c.out, "Pinned the loader to %s.\n", value)
		})
	default:
		return usageError{"unknown loader command " + c.args[1]}
	}
}
