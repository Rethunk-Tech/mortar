package installer

import (
	"path"
	"slices"
)

var _ = Register(plain{})

// plain is the folder-per-package layout: everything the archive holds, minus junk, goes in the mods folder under the
// package's key.
type plain struct{}

func (plain) ID() string { return "plain" }

func (plain) Detect(Archive, Game) bool { return true }

func (plain) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	all, err := files(a, ".")
	if err != nil {
		return Layout{}, err
	}
	// A folder game's mods are loose files the game reads below the target, so no folder is made for the archive.
	folder := slices.Contains(g.Loaders, "folder")
	var l Layout
	for _, f := range all {
		if !skip(f) {
			rel := path.Join(a.Key, f)
			if folder {
				rel = f
			}
			l.Files = append(l.Files, File{Src: f, Target: TargetMods, Rel: rel})
		}
	}
	return l, validate(l, g)
}
