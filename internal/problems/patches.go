package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

// patchHub is how many mods replacing one method make it a shared hook point rather than a sign of overlap.
const patchHub = 4

// withPatches adds the "patches" hints from the bridge's last startup report in the profile folder.
func (s *Service) withPatches(gameID, id string, r Result, mods []Installed) Result {
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return r
	}
	if hints := patchHints(launchsvc.LatestReplaces(dir), mods); len(hints) > 0 {
		r.Redundant = slices.Concat(r.Redundant, hints)
	}
	return r
}

// patchHints flags enabled mods that replace largely the same game methods as another enabled mod: at least two
// shared methods covering half or more of the smaller mod's set. Methods four or more mods replace are left out of
// the shared count. replaces is keyed by Harmony ID, which mods set to their UniqueID by convention.
func patchHints(replaces map[string][]string, mods []Installed) []Redundant {
	byID := map[string][]string{}
	for owner, methods := range replaces {
		byID[strings.ToLower(owner)] = methods
	}
	var patched []Installed
	sets := map[string]map[string]bool{}
	seen := map[string]bool{}
	for _, m := range mods {
		id := strings.ToLower(m.UniqueID)
		methods := byID[id]
		if !m.Enabled || len(methods) == 0 || seen[id] {
			continue
		}
		seen[id] = true
		set := map[string]bool{}
		for _, method := range methods {
			set[method] = true
		}
		patched = append(patched, m)
		sets[m.Key] = set
	}
	count := map[string]int{}
	for _, set := range sets {
		for method := range set {
			count[method]++
		}
	}
	by := map[string][]ModRef{}
	shared := map[string]map[string]bool{}
	for i, a := range patched {
		for _, b := range patched[i+1:] {
			var both []string
			for method := range sets[a.Key] {
				if sets[b.Key][method] && count[method] < patchHub {
					both = append(both, method)
				}
			}
			if len(both) < 2 || 2*len(both) < min(len(sets[a.Key]), len(sets[b.Key])) {
				continue
			}
			by[a.Key] = append(by[a.Key], ModRef{Key: b.Key, Name: b.Name})
			by[b.Key] = append(by[b.Key], ModRef{Key: a.Key, Name: a.Name})
			for _, k := range []string{a.Key, b.Key} {
				if shared[k] == nil {
					shared[k] = map[string]bool{}
				}
				for _, method := range both {
					shared[k][method] = true
				}
			}
		}
	}
	var out []Redundant
	for _, m := range patched {
		if len(by[m.Key]) > 0 {
			out = append(out, Redundant{Kind: "patches", Key: m.Key, UniqueID: m.UniqueID, Name: m.Name, By: by[m.Key], Detail: shortMethods(shared[m.Key])})
		}
	}
	return out
}

// shortMethods names up to three methods as "Type.Method", from the bridge's "Namespace.Type::Method".
func shortMethods(methods map[string]bool) string {
	var names []string
	for method := range methods {
		typ, name, _ := strings.Cut(method, "::")
		typ = typ[strings.LastIndexAny(typ, ".+")+1:]
		names = append(names, typ+"."+name)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + ", …"
	}
	return strings.Join(names, ", ")
}
