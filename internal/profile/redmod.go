package profile

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// cyberpunkFile is a file of a profile's enabled packages, as a path in the game folder, and how far down the
// profile its package is: a higher rank is a later entry, which wins a conflict.
type cyberpunkFile struct {
	rel  string
	rank int
}

// cyberpunkFiles are the files the profile's enabled packages lay out for a game whose mods target is the Cyberpunk
// layout's; none for any other game.
func (s *Store) cyberpunkFiles(game, id string) ([]cyberpunkFile, error) {
	info, ok := components.Game(game)
	if t, has := info.Target(installer.TargetMods); !ok || !has || t.Layout != components.LayoutCyberpunk {
		return nil, nil
	}
	owners, err := s.PackageFileOwners(game, id)
	if err != nil {
		return nil, err
	}
	p, _, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	rank := map[string]int{}
	for i, e := range p.Entries {
		rank[e.Key] = i
	}
	t, _ := info.Target(installer.TargetMods)
	prefix := t.ProfileFolder()
	var out []cyberpunkFile
	for rel, key := range owners {
		if prefix != "" {
			var below bool
			if rel, below = strings.CutPrefix(rel, prefix+"/"); !below {
				continue
			}
		}
		out = append(out, cyberpunkFile{path.Clean(rel), rank[key]})
	}
	return out, nil
}

// REDmodFolders are the folder names of the REDmods the profile's enabled packages lay out, in the order of REDmod's
// modlist: the first mod wins a file (Vortex's cyberpunk2077_ext_redux load_order.ts), so the profile's last entry,
// which wins here, comes first. A game whose mods target is not a REDmod one has none.
func (s *Store) REDmodFolders(game, id string) ([]string, error) {
	files, err := s.cyberpunkFiles(game, id)
	if err != nil {
		return nil, err
	}
	type mod struct {
		name string
		at   int
	}
	var found []mod
	for _, f := range files {
		segs := strings.Split(f.rel, "/")
		if len(segs) == 3 && strings.EqualFold(segs[0], "mods") && strings.EqualFold(segs[2], "info.json") {
			found = append(found, mod{segs[1], f.rank})
		}
	}
	slices.SortFunc(found, func(a, b mod) int { return cmp.Or(cmp.Compare(b.at, a.at), cmp.Compare(a.name, b.name)) })
	var out []string
	for _, m := range found {
		out = append(out, m.name)
	}
	return out, nil
}

// ArchiveOrder are the names of the .archive files directly in archive/pc/mod, in the order the profile loads them: the
// game loads a listed archive before the ones after it, so the profile's last entry, which wins, comes last, and an
// entry's own archives are in name order.
func (s *Store) ArchiveOrder(game, id string) ([]string, error) {
	files, err := s.cyberpunkFiles(game, id)
	if err != nil {
		return nil, err
	}
	type archive struct {
		name string
		at   int
	}
	var found []archive
	for _, f := range files {
		dir, name := path.Split(f.rel)
		if strings.EqualFold(dir, "archive/pc/mod/") && strings.EqualFold(path.Ext(name), ".archive") {
			found = append(found, archive{name, f.rank})
		}
	}
	slices.SortFunc(found, func(a, b archive) int { return cmp.Or(cmp.Compare(a.at, b.at), cmp.Compare(a.name, b.name)) })
	var out []string
	for _, a := range found {
		out = append(out, a.name)
	}
	return out, nil
}
