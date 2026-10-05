// Package loadorder lists enabled mods in the order SMAPI's ModResolver loads them.
//
// SMAPI walks required dependencies, optional dependencies, and ContentPackFor as
// predecessors, so those mods load before anything that names them; remaining ties
// are alphabetical by display name (then mod id). Mortar does not change that order.
package loadorder

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Mod is the manifest fields the resolver needs.
type Mod struct {
	ID             mod.ID
	Name           string
	Needs          []mod.ID
	Optional       []mod.ID
	ContentPackFor mod.ID
}

// Row is one enabled mod in SMAPI load order.
type Row struct {
	Position        int      `json:"position"`
	ID              mod.ID   `json:"id"`
	Name            string   `json:"name"`
	Required        []mod.ID `json:"required,omitempty"`
	Optional        []mod.ID `json:"optional,omitempty"`
	Dependents      []mod.ID `json:"dependents,omitempty"`
	MissingRequired []mod.ID `json:"missingRequired,omitempty"`
	Cycle           bool     `json:"cycle,omitempty"`
}

type node struct {
	id       mod.ID
	mod      Mod
	required []mod.ID
	optional []mod.ID
	preds    []string
}

func cleanIDs(ids []mod.ID) []mod.ID {
	out := make([]mod.ID, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id.Fold()] {
			continue
		}
		seen[id.Fold()] = true
		out = append(out, id)
	}
	return out
}

func byName(a, b node) int {
	if n := cmp.Compare(strings.ToLower(a.mod.Name), strings.ToLower(b.mod.Name)); n != 0 {
		return n
	}
	return cmp.Compare(a.id.Fold(), b.id.Fold())
}

// Resolve returns enabled mods in SMAPI load order. Duplicate mod ids keep the first.
func Resolve(mods []Mod) []Row {
	nodes := make([]node, 0, len(mods))
	index := map[string]int{}
	for _, m := range mods {
		id := m.ID
		if id == "" {
			continue
		}
		key := id.Fold()
		if _, ok := index[key]; ok {
			continue
		}
		required := cleanIDs(m.Needs)
		if cp := m.ContentPackFor; cp != "" && !slices.ContainsFunc(required, func(x mod.ID) bool { return x.Fold() == cp.Fold() }) {
			required = append(required, cp)
		}
		optional := cleanIDs(m.Optional)
		optional = slices.DeleteFunc(optional, func(x mod.ID) bool {
			return slices.ContainsFunc(required, func(y mod.ID) bool { return x.Fold() == y.Fold() })
		})
		index[key] = len(nodes)
		nodes = append(nodes, node{id: id, mod: m, required: required, optional: optional})
	}
	present := func(id mod.ID) bool {
		_, ok := index[id.Fold()]
		return ok
	}
	for i := range nodes {
		n := &nodes[i]
		preds := make([]string, 0, len(n.required)+len(n.optional))
		for _, id := range n.required {
			if present(id) {
				preds = append(preds, id.Fold())
			}
		}
		for _, id := range n.optional {
			if present(id) {
				preds = append(preds, id.Fold())
			}
		}
		n.preds = preds
	}

	incoming := make([]int, len(nodes))
	graph := make([][]int, len(nodes))
	for i, n := range nodes {
		seen := map[string]bool{}
		for _, pred := range n.preds {
			if seen[pred] {
				continue
			}
			seen[pred] = true
			incoming[i]++
			p := index[pred]
			graph[p] = append(graph[p], i)
		}
	}

	ready := make([]int, 0, len(nodes))
	for i, n := range incoming {
		if n == 0 {
			ready = append(ready, i)
		}
	}
	sortReady := func() {
		slices.SortFunc(ready, func(a, b int) int { return byName(nodes[a], nodes[b]) })
	}
	sortReady()

	ordered := make([]int, 0, len(nodes))
	placed := make([]bool, len(nodes))
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]
		if placed[i] {
			continue
		}
		placed[i] = true
		ordered = append(ordered, i)
		for _, j := range graph[i] {
			incoming[j]--
			if incoming[j] == 0 {
				ready = append(ready, j)
				sortReady()
			}
		}
	}

	cycle := map[int]bool{}
	rest := make([]int, 0)
	for i := range nodes {
		if !placed[i] {
			cycle[i] = true
			rest = append(rest, i)
		}
	}
	slices.SortFunc(rest, func(a, b int) int { return byName(nodes[a], nodes[b]) })
	ordered = append(ordered, rest...)

	dependents := make([][]mod.ID, len(nodes))
	for _, n := range nodes {
		for _, pred := range n.preds {
			p := index[pred]
			dependents[p] = append(dependents[p], n.id)
		}
	}
	for i := range dependents {
		slices.SortFunc(dependents[i], func(a, b mod.ID) int {
			return byName(nodes[index[a.Fold()]], nodes[index[b.Fold()]])
		})
	}

	out := make([]Row, 0, len(ordered))
	for pos, i := range ordered {
		n := nodes[i]
		missing := make([]mod.ID, 0)
		for _, id := range n.required {
			if !present(id) {
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
