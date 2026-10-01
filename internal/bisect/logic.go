// Package bisect finds a crashing mod by testing dependency-aware subsets.
package bisect

import (
	"context"
	"errors"
	"slices"
	"sort"
)

// Mod is the dependency graph used by the crash bisector.
type Mod struct {
	ID           string
	Dependencies []string
	Group        string
}

type Progress struct {
	Step     int
	Total    int
	ModsLeft int
}

// Split divides the candidate list into two non-empty halves when possible.
func Split(mods []Mod) (left, right []Mod) {
	if len(mods) < 2 {
		return append([]Mod(nil), mods...), nil
	}
	mid := (len(mods) + 1) / 2
	return append([]Mod(nil), mods[:mid]...), append([]Mod(nil), mods[mid:]...)
}

func dependencyGroups(mods []Mod) [][]Mod {
	parent := make(map[string]string, len(mods))
	known := make(map[string]Mod, len(mods))
	for _, mod := range mods {
		parent[mod.ID] = mod.ID
		known[mod.ID] = mod
	}
	var find func(string) string
	find = func(id string) string {
		root := parent[id]
		if root != id {
			parent[id] = find(root)
		}
		return parent[id]
	}
	union := func(a, b string) {
		if _, ok := parent[a]; !ok {
			return
		}
		if _, ok := parent[b]; !ok {
			return
		}
		a, b = find(a), find(b)
		if a != b {
			parent[b] = a
		}
	}
	groups := make(map[string]string)
	for _, mod := range mods {
		if mod.Group == "" {
			continue
		}
		if first, ok := groups[mod.Group]; ok {
			union(first, mod.ID)
		} else {
			groups[mod.Group] = mod.ID
		}
	}
	for _, mod := range mods {
		for _, dependency := range mod.Dependencies {
			union(mod.ID, dependency)
		}
	}
	byRoot := make(map[string][]Mod)
	order := make([]string, 0, len(mods))
	for _, mod := range mods {
		root := find(mod.ID)
		if _, ok := byRoot[root]; !ok {
			order = append(order, root)
		}
		byRoot[root] = append(byRoot[root], known[mod.ID])
	}
	out := make([][]Mod, 0, len(order))
	for _, root := range order {
		out = append(out, byRoot[root])
	}
	return out
}

func find(ctx context.Context, mods []Mod, run func(context.Context, []string) (bool, error), report func(Progress)) ([]Mod, error) {
	groups := dependencyGroups(mods)
	if len(groups) == 0 {
		return nil, errors.New("no user mods to test")
	}
	total := 0
	for n := len(groups); n > 1; n = (n + 1) / 2 {
		total++
	}
	candidates := append([][]Mod(nil), groups...)
	for step := 1; len(candidates) > 1; step++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		mid := (len(candidates) + 1) / 2
		left := candidates[:mid]
		right := candidates[mid:]
		seeds := make([]string, 0)
		for _, group := range left {
			for _, mod := range group {
				seeds = append(seeds, mod.ID)
			}
		}
		disabled := DisableClosure(mods, seeds)
		if report != nil {
			report(Progress{Step: step, Total: total, ModsLeft: len(flatten(right))})
		}
		failed, err := run(ctx, disabled)
		if err != nil {
			return nil, err
		}
		if failed {
			candidates = right
		} else {
			candidates = left
		}
	}
	return append([]Mod(nil), candidates[0]...), nil
}

func flatten(groups [][]Mod) []Mod {
	var mods []Mod
	for _, group := range groups {
		mods = append(mods, group...)
	}
	return mods
}

// DisableClosure returns every mod disabled by the seed mods, including their
// grouped framework members and all transitive dependents.
func DisableClosure(mods []Mod, seeds []string) []string {
	known := make(map[string]Mod, len(mods))
	groups := make(map[string][]string)
	for _, mod := range mods {
		known[mod.ID] = mod
		if mod.Group != "" {
			groups[mod.Group] = append(groups[mod.Group], mod.ID)
		}
	}

	disabled := make(map[string]bool, len(seeds))
	pending := append([]string(nil), seeds...)
	for len(pending) > 0 {
		id := pending[0]
		pending = pending[1:]
		if disabled[id] {
			continue
		}
		mod, ok := known[id]
		if !ok {
			continue
		}
		disabled[id] = true
		if mod.Group != "" {
			pending = append(pending, groups[mod.Group]...)
		}
		for _, candidate := range mods {
			if slices.Contains(candidate.Dependencies, id) {
				pending = append(pending, candidate.ID)
			}
		}
	}

	result := make([]string, 0, len(disabled))
	for _, mod := range mods {
		if disabled[mod.ID] {
			result = append(result, mod.ID)
		}
	}
	sort.Strings(result)
	return result
}
