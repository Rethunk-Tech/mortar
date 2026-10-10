package installer

import (
	"path"
	"slices"
	"strings"
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
			rel, target := path.Join(a.Key, f), TargetMods
			if folder {
				rel = f
				ext := strings.ToLower(strings.TrimPrefix(path.Ext(f), "."))
				if i := slices.IndexFunc(g.Targets, func(t Target) bool { return slices.Contains(t.Extensions, ext) }); i >= 0 {
					target = g.Targets[i].ID
				}
			}
			l.Files = append(l.Files, File{Src: f, Target: target, Rel: rel})
		}
	}
	return l, validate(l, g)
}
