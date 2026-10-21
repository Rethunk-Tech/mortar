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
	// A placed mods target's mods are loose files the game reads below it, so no folder is made for the archive,
	// unless the archive holds a mod folder of its own.
	folder := g.Placed
	if file, ok := modRootFile(all, g); folder && ok {
		return wholeFolder(a, g, all, path.Dir(file))
	}
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

// MatchesModRoot reports whether a file's base name is the mods target's ModRoot, a case-insensitive path.Match
// pattern: "*" makes every archive one folder, its own top folder or the archive root.
func MatchesModRoot(name, root string) bool {
	ok, err := path.Match(strings.ToLower(root), strings.ToLower(name))
	return err == nil && ok
}

// ModRootFile is the slash path in the archive of the shallowest file named as the mods target's ModRoot.
func ModRootFile(a Archive, g Game) (string, bool) {
	all, err := files(a, ".")
	if err != nil {
		return "", false
	}
	return modRootFile(all, g)
}

func modRootFile(all []string, g Game) (string, bool) {
	ti := slices.IndexFunc(g.Targets, func(t Target) bool { return t.ID == TargetMods })
	if ti < 0 || g.Targets[ti].ModRoot == "" {
		return "", false
	}
	depth := func(d string) int {
		if d == "." {
			return -1
		}
		return strings.Count(d, "/")
	}
	best, found := "", false
	for _, f := range all {
		if skip(f) || !MatchesModRoot(path.Base(f), g.Targets[ti].ModRoot) {
			continue
		}
		if !found || depth(path.Dir(f)) < depth(path.Dir(best)) {
			best, found = f, true
		}
	}
	return best, found
}

// wholeFolder lays the mod folder dir out as one folder named after the package; files outside it are not the mod's.
func wholeFolder(a Archive, g Game, all []string, dir string) (Layout, error) {
	var l Layout
	for _, f := range all {
		rel := f
		if dir != "." {
			var ok bool
			if rel, ok = strings.CutPrefix(f, dir+"/"); !ok {
				continue
			}
		}
		if !skip(f) {
			l.Files = append(l.Files, File{Src: f, Target: TargetMods, Rel: path.Join(a.Key, rel)})
		}
	}
	return l, validate(l, g)
}
