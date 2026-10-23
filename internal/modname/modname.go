// Package modname names a mod folder from its files the way the game's loader reads it, so a screen that lists
// folders of any game does not read every one as a SMAPI mod.
package modname

import (
	"cmp"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// unknownPublisher names a package whose manifest has no publisher, as profile does.
const unknownPublisher = "Unknown"

// Info is what a mod folder's files say of it; the zero Info means they say nothing.
type Info struct {
	ID      mod.ID
	Name    string
	Version string
}

// Of reads the mod in dir: a SMAPI game's first manifest.json, a package game's Thunderstore manifest or else its
// plugin DLL. A game with loose-file mods gives the zero Info.
func Of(gameID, dir string) Info {
	g := installer.GameOf(gameID)
	switch {
	case slices.Contains(g.Loaders, smapi.ID):
		mods, err := smapimanifest.Scan(dir)
		if err != nil || len(mods) == 0 {
			return Info{}
		}
		return Info{ID: mods[0].ModID(), Name: mods[0].Name, Version: mods[0].Version}
	case installer.TakesPackages(g):
		return packageInfo(dir)
	}
	return Info{}
}

func packageInfo(dir string) Info {
	root := installer.Open(dir, "").PackageRoot()
	if b, err := fsx.ReadFile(filepath.Join(root.Dir, "manifest.json")); err == nil {
		if m, err := bepinex5.ParseManifest(b); err == nil && m.Name != "" {
			publisher := cmp.Or(m.Namespace, m.Author, unknownPublisher)
			return Info{ID: mod.NewID(mod.FormatThunderstore, publisher+"-"+m.Name), Name: m.Name, Version: m.Version}
		}
	}
	var dlls []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".dll") {
			dlls = append(dlls, p)
		}
		return nil
	})
	if len(dlls) == 0 {
		return Info{}
	}
	slices.Sort(dlls)
	name := strings.TrimSuffix(filepath.Base(dlls[0]), filepath.Ext(dlls[0]))
	return Info{ID: mod.NewID(mod.FormatBepInEx, name), Name: name}
}
