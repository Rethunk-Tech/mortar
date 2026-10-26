package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func (c *cmd) dev() error {
	if len(c.args) < 2 {
		return usageError{"dev needs add or clone"}
	}
	var method string
	var p control.Params
	switch c.args[1] {
	case "add":
		a, err := c.need(2, "a game", "a folder")
		if err != nil {
			return err
		}
		dir, err := filepath.Abs(a[1])
		if err != nil {
			return err
		}
		method, p = "dev.add", control.Params{Game: a[0], Path: dir}
	case "clone":
		a, err := c.need(2, "a game", "a repository")
		if err != nil {
			return err
		}
		parent := c.dirFlag
		if parent == "" {
			if parent, err = os.Getwd(); err != nil {
				return err
			}
		}
		if parent, err = filepath.Abs(parent); err != nil {
			return err
		}
		method, p = "dev.clone", control.Params{Game: a[0], ID: a[1], Path: parent}
	default:
		return usageError{"dev needs add or clone"}
	}
	p.Profile = c.profileFlag
	var project settings.DevProject
	if err := c.call(method, p, &project, installTimeout); err != nil {
		return err
	}
	return c.emit(project, func() {
		fmt.Fprintf(c.out, "Added %s (%s).\n", project.Name, project.Dir)
	})
}
