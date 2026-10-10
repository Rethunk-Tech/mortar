// Package folder is the loader of a game that has none: its mods are files in a folder the game reads, which the
// deployer fills from the profile for the length of a launch and empties again.
package folder

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// ID is the catalog's id for this loader.
const ID = "folder"

// ModsRole is the catalog path role of the folder the game reads its mods from.
const ModsRole = "mods"

// TargetID is the catalog target whose files the loader places.
const TargetID = "mods"

var _ = loader.Register(Loader{})

// Loader has nothing to install: the folder exists for as long as the game does.
type Loader struct{}

func (Loader) ID() string { return ID }

func (Loader) Formats() []string { return nil }

func (Loader) Builtin() {}

func (Loader) Status(loader.Target) (loader.Status, error) {
	return loader.Status{Installed: true}, nil
}

func (Loader) Install(context.Context, loader.Target, loader.Package, func(loader.Step)) (string, error) {
	return "", errors.New("this game has no mod loader to install")
}

// Contribute places every file the profile's enabled mods laid out in the mods target's folder into the game's mods
// folder. Files the player already has there are the deployer's to displace and give back, never the loader's.
func (Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	info, ok := components.Game(p.Game)
	if !ok {
		return errors.New("unknown game " + p.Game)
	}
	if _, has := info.Paths[ModsRole]; !has {
		return errors.New(info.Name + " names no " + ModsRole + " folder")
	}
	t, ok := info.Target(TargetID)
	if !ok {
		return errors.New(info.Name + " has no " + TargetID + " target")
	}
	root := filepath.Join(p.Dir, filepath.FromSlash(t.ProfileFolder()))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == root {
				return nil
			}
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		plan.AddFile(launchplan.PlanFile{Src: path, Dst: rel, Root: ModsRole})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
