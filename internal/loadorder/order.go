// Package loadorder is a profile's load order as its loader computes it. Resolve is SMAPI's: ModResolver.ProcessDependencies
// (Pathoschild/SMAPI develop e5b8bf3, src/SMAPI/Framework/ModLoading/ModResolver.cs:221-391) walks the mods in the
// order its scanner found their folders and loads each mod's dependencies, required, optional and the mod a content
// pack is for, just before it, depth first. Mortar does not change that order.
package loadorder

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Dep is one dependency of a manifest, in the order the manifest declares it.
type Dep struct {
	ID       mod.ID
	Required bool
}

// Mod is the manifest fields the resolver needs. Folder is the mod's folder, whose name orders the walk.
type Mod struct {
	ID             mod.ID
	Name           string
	Folder         string
	Deps           []Dep
	ContentPackFor mod.ID
}

// Row is one enabled mod, or plugin, in its loader's load order.
type Row struct {
	Position        int      `json:"position"`
	ID              mod.ID   `json:"id"`
	Name            string   `json:"name"`
	Required        []mod.ID `json:"required,omitempty"`
	Optional        []mod.ID `json:"optional,omitempty"`
	Dependents      []mod.ID `json:"dependents,omitempty"`
	MissingRequired []mod.ID `json:"missingRequired,omitempty"`
	Cycle           bool     `json:"cycle,omitempty"`
	// LastRun is a mod the loader's last run loaded at this place rather than where its dependencies predict it.
	LastRun bool `json:"lastRun,omitempty"`
}

type node struct {
	id       mod.ID
	mod      Mod
	deps     []Dep
	required []mod.ID
	optional []mod.ID
}

// byFolder is the order SMAPI's scanner meets mod folders on Windows: folder by folder, names without regard to case.
// Mods without a folder go by name, then id.
func byFolder(a, b node) int {
	sa := strings.Split(strings.ToLower(filepath.ToSlash(a.mod.Folder)), "/")
	sb := strings.Split(strings.ToLower(filepath.ToSlash(b.mod.Folder)), "/")
	if n := slices.Compare(sa, sb); n != 0 {
		return n
	}
	if n := cmp.Compare(strings.ToLower(a.mod.Name), strings.ToLower(b.mod.Name)); n != 0 {
		return n
	}
	return cmp.Compare(a.id.Fold(), b.id.Fold())
}

type state int

const (
	queued state = iota
	checking
	sorted
	failed
)

// Resolve returns enabled mods in SMAPI load order. Duplicate mod ids keep the first. A mod that misses a required
// dependency, or that a dependency loop or a failed required dependency stops, is placed where SMAPI gives up on it.
func Resolve(mods []Mod) []Row {
	nodes := make([]node, 0, len(mods))
	index := map[string]int{}
	for _, m := range mods {
		if m.ID == "" {
			continue
		}
		if _, ok := index[m.ID.Fold()]; ok {
			continue
		}
		n := node{id: m.ID, mod: m}
		seen := map[string]bool{}
		for _, d := range m.Deps {
			if d.ID != "" && !seen[d.ID.Fold()] {
				seen[d.ID.Fold()] = true
				n.deps = append(n.deps, d)
			}
		}
		// SMAPI lists the content pack's parent after the manifest's dependencies, always required.
		if cp := m.ContentPackFor; cp != "" {
			if i := slices.IndexFunc(n.deps, func(d Dep) bool { return d.ID.Fold() == cp.Fold() }); i >= 0 {
				n.deps[i].Required = true
			} else {
				n.deps = append(n.deps, Dep{ID: cp, Required: true})
			}
		}
		for _, d := range n.deps {
			if d.Required {
				n.required = append(n.required, d.ID)
			} else {
				n.optional = append(n.optional, d.ID)
			}
		}
		index[m.ID.Fold()] = len(nodes)
		nodes = append(nodes, n)
	}
	find := func(id mod.ID) (int, bool) {
		i, ok := index[id.Fold()]
		return i, ok
	}
	walk := make([]int, len(nodes))
	for i := range walk {
		walk[i] = i
	}
	slices.SortStableFunc(walk, func(a, b int) int { return byFolder(nodes[a], nodes[b]) })

	states := make([]state, len(nodes))
	cycle := make([]bool, len(nodes))
	ordered := make([]int, 0, len(nodes))
	var visit func(i int) state
	visit = func(i int) state {
		if states[i] != queued {
			return states[i]
		}
		n := nodes[i]
		for _, d := range n.deps {
			if _, ok := find(d.ID); !ok && d.Required {
				ordered = append(ordered, i)
				states[i] = failed
				return failed
			}
		}
		states[i] = checking
		for _, d := range n.deps {
			j, ok := find(d.ID)
			if !ok {
				continue
			}
			if states[j] == checking {
				ordered = append(ordered, i)
				cycle[i] = true
				states[i] = failed
				return failed
			}
			if visit(j) == failed && d.Required {
				ordered = append(ordered, i)
				cycle[i] = cycle[j]
				states[i] = failed
				return failed
			}
		}
		ordered = append(ordered, i)
		states[i] = sorted
		return sorted
	}
	for _, i := range walk {
		visit(i)
	}

	dependents := make([][]mod.ID, len(nodes))
	for _, n := range nodes {
		for _, d := range n.deps {
			if p, ok := find(d.ID); ok {
				dependents[p] = append(dependents[p], n.id)
			}
		}
	}
	for i := range dependents {
		slices.SortFunc(dependents[i], func(a, b mod.ID) int {
			return byFolder(nodes[index[a.Fold()]], nodes[index[b.Fold()]])
		})
	}

	out := make([]Row, 0, len(ordered))
	for pos, i := range ordered {
		n := nodes[i]
		missing := make([]mod.ID, 0)
		for _, id := range n.required {
			if _, ok := find(id); !ok {
				missing = append(missing, id)
			}
		}
		out = append(out, Row{
			Position:        pos + 1,
			ID:              n.id,
			Name:            n.mod.Name,
			Required:        n.required,
			Optional:        n.optional,
			Dependents:      dependents[i],
			MissingRequired: missing,
			Cycle:           cycle[i],
		})
	}
	return out
}
