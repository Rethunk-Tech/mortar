package profile

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// REDmodFolders are the folder names of the REDmods the profile's enabled packages lay out, in the order of REDmod's
// modlist: the first mod wins a file (Vortex's cyberpunk2077_ext_redux load_order.ts), so the profile's last entry,
// which wins here, comes first. A game whose mods target is not a REDmod one has none.
func (s *Store) REDmodFolders(game, id string) ([]string, error) {
	info, ok := components.Game(game)
	if t, has := info.Target(installer.TargetMods); !ok || !has || !t.REDmod {
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
	type mod struct {
		name string
		at   int
	}
	var found []mod
	for rel, key := range owners {
		if prefix != "" {
			var below bool
			if rel, below = strings.CutPrefix(rel, prefix+"/"); !below {
				continue
			}
		}
		segs := strings.Split(path.Clean(rel), "/")
		if len(segs) == 3 && strings.EqualFold(segs[0], "mods") && strings.EqualFold(segs[2], "info.json") {
			found = append(found, mod{segs[1], rank[key]})
		}
	}
	slices.SortFunc(found, func(a, b mod) int { return cmp.Or(cmp.Compare(b.at, a.at), cmp.Compare(a.name, b.name)) })
	out := make([]string, len(found))
	for i, m := range found {
		out[i] = m.name
	}
	return out, nil
}
