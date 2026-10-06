package dotnet

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// Loaded is a plugin in BepInEx's load order with the dependencies of the copy that loads.
type Loaded struct {
	Plugin
	Deps []Relation
	// Cycle is part of a dependency cycle, on which BepInEx throws and loads no plugin at all.
	Cycle bool
}

// LoadOrder is the order BepInEx 5's chainloader loads plugins in (Chainloader.Start and Utility.TopologicalSort,
// BepInEx 5.4.23). Of several copies of one GUID only the newest version loads. The GUIDs are walked sorted without
// regard to case, and each plugin's dependencies, hard and soft, go before it, depth first, in the order its attributes
// declare them; that walk is the only tie-break. assemblies holds one Declared per DLL, so the dependencies are those of
// the copy that loads.
//
// ponytail: GUIDs sort by lower-cased ordinal where BepInEx uses StringComparer.InvariantCultureIgnoreCase; they differ
// only on punctuation such as '-' against letters, which no GUID in a 77-plugin real run hit.
func LoadOrder(assemblies []Declared) []Loaded {
	newest := map[string]Loaded{}
	for _, d := range assemblies {
		for _, p := range d.Plugins {
			key := strings.ToLower(p.GUID)
			if have, ok := newest[key]; ok {
				if c, ok := meta.CompareVersions(p.Version, have.Version); !ok || c <= 0 {
					continue
				}
			}
			l := Loaded{Plugin: p}
			for _, r := range d.Relations {
				if r.Plugin == p.GUID && r.Kind != Incompatible {
					l.Deps = append(l.Deps, r)
				}
			}
			newest[key] = l
		}
	}
	keys := make([]string, 0, len(newest))
	for k := range newest {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var order []string
	cycle := map[string]bool{}
	done := map[string]bool{}
	var stack []string
	var visit func(k string)
	visit = func(k string) {
		if i := slices.Index(stack, k); i >= 0 {
			for _, c := range stack[i:] {
				cycle[c] = true
			}
			return
		}
		if done[k] {
			return
		}
		stack = append(stack, k)
		for _, dep := range newest[k].Deps {
			visit(strings.ToLower(dep.GUID))
		}
		stack = stack[:len(stack)-1]
		done[k] = true
		if _, ok := newest[k]; ok {
			order = append(order, k)
		}
	}
	for _, k := range keys {
		visit(k)
	}
	out := make([]Loaded, len(order))
	for i, k := range order {
		out[i] = newest[k]
		out[i].Cycle = cycle[k]
	}
	return out
}
