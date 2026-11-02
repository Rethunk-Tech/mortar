// Package folder is the loader of a game that has none: its mods are files in a folder the game reads, which the
// deployer fills from the profile for the length of a launch and empties again.
package folder

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

// Extension is per-launch work a game's catalog asks of the folder loader by naming a path role: the game's files
// outside its mods folder that the loader also fills or writes.
type Extension interface {
	// Role is the catalog path role that switches the extension on for a game.
	Role() string
	// Contribute adds the extension's files to plan; it has no side effects.
	Contribute(plan *launchplan.Plan, p loader.ProfileView)
	// Prelaunch writes what Contribute placed, for a launch actually starting. p.Enabled is the profile's enabled
	// package folders in mod order.
	Prelaunch(p loader.ProfileView) error
}

var extensions []Extension

// Extend adds an extension and reports true, for a driver package's blank variable like loader.Register.
func Extend(e Extension) bool {
	extensions = append(extensions, e)
	return true
}

func extensionsOf(gameID string) []Extension {
	info, ok := components.Game(gameID)
	if !ok {
		return nil
	}
	var out []Extension
	for _, e := range extensions {
		if _, has := info.Paths[e.Role()]; has {
			out = append(out, e)
		}
	}
	return out
}

// WantsOrder reports whether the game has an extension that needs ProfileView.Enabled.
func WantsOrder(gameID string) bool { return len(extensionsOf(gameID)) > 0 }

// Prelaunch runs the game's extensions.
func (Loader) Prelaunch(p loader.ProfileView) error {
	for _, e := range extensionsOf(p.Game) {
		if err := e.Prelaunch(p); err != nil {
			return err
		}
	}
	return nil
}

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

// modInfoFile is the mod-root file whose modid a profile records as the mod's own id.
const modInfoFile = "modinfo.json"

// ModIDName is the term for the modid a game's modinfo.json gives; a game whose mod-root file is another format (or
// that has none) has mods with no id of their own.
func (Loader) ModIDName(gameID string) string {
	if info, ok := components.Game(gameID); ok {
		if t, ok := info.Target(TargetID); ok && strings.EqualFold(t.ModRoot, modInfoFile) {
			return "ModID"
		}
	}
	return ""
}

// PlacedFolder is the mods target's folder in the profile, whose files Contribute places in the game's mods folder.
func (Loader) PlacedFolder(gameID string) (string, string, bool) {
	info, ok := components.Game(gameID)
	if !ok {
		return "", "", false
	}
	if _, has := info.Paths[ModsRole]; !has {
		return "", "", false
	}
	t, ok := info.Target(TargetID)
	if !ok {
		return "", "", false
	}
	return t.ProfileFolder(), ModsRole, true
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
	for _, e := range extensionsOf(p.Game) {
		e.Contribute(plan, p)
	}
	return nil
}
