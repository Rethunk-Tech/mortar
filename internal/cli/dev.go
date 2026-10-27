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
		return usageError{"dev needs add, clone, list or remove"}
	}
	switch c.args[1] {
	case "list":
		return c.devList()
	case "remove":
		return c.devRemove()
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
		return usageError{"dev needs add, clone, list or remove"}
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

func (c *cmd) devList() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	var projects []settings.DevProject
	if err := c.call("dev.list", control.Params{Game: a[0]}, &projects, readTimeout); err != nil {
		return err
	}
	return c.emit(projects, func() {
		for _, p := range projects {
			fmt.Fprintf(c.out, "%s\t%s\n", p.Name, p.Dir)
		}
	})
}

// devRemove takes a project off the Develop tab by its folder or its name; the folder is not touched.
func (c *cmd) devRemove() error {
	a, err := c.need(2, "a game", "a project folder or name")
	if err != nil {
		return err
	}
	dir, err := filepath.Abs(a[1])
	if err != nil {
		return err
	}
	var project settings.DevProject
	if err := c.call("dev.remove", control.Params{Game: a[0], Path: dir, ID: a[1]}, &project, readTimeout); err != nil {
		return err
	}
	return c.emit(project, func() {
		fmt.Fprintf(c.out, "Removed %s from the Develop tab; %s is unchanged.\n", project.Name, project.Dir)
	})
}
