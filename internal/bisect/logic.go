// Package bisect finds a crashing mod by testing dependency-aware subsets.
package bisect

import (
	"context"
	"errors"
	"slices"
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

// ErrNoCrash is a check whose first run, with every mod on, did not crash: there is nothing to narrow down.
var ErrNoCrash = errors.New("the game did not crash with every mod on, so no mod is to blame")

// find narrows the mods down to the one the crash needs. Each step switches off half of the suspects and every mod
// that needs them, never a mod something still on needs: suspects nothing else among them needs go first, so a
// library is tested only once its dependents are cleared. A crash with them off clears them; no crash convicts them.
// Mods that always switch together (one package) are found together.
func find(ctx context.Context, mods []Mod, run func(context.Context, []string) (bool, error), report func(Progress)) ([]Mod, error) {
	if len(mods) == 0 {
		return nil, errors.New("no user mods to test")
	}
	crashed, err := run(ctx, nil)
	if err != nil {
		return nil, err
	}
	if !crashed {
		return nil, ErrNoCrash
	}
	suspects := make([]string, len(mods))
	for i, m := range mods {
		suspects[i] = m.ID
	}
	total := progressSteps(len(suspects))
	for step := 1; len(suspects) > 1; step++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		off := split(mods, suspects)
		if off == nil {
			break
		}
		if report != nil {
			report(Progress{Step: step, Total: max(total, step), ModsLeft: len(suspects)})
		}
		failed, err := run(ctx, off)
		if err != nil {
			return nil, err
		}
		// Still crashing with them off clears them; running clean convicts one of them.
		suspects = slices.DeleteFunc(suspects, func(id string) bool { return slices.Contains(off, id) == failed })
	}
	found := make([]Mod, 0, len(suspects))
	for _, m := range mods {
		if slices.Contains(suspects, m.ID) {
			found = append(found, m)
		}
	}
	return found, nil
}

// split is the mods to switch off for the next step: half of the suspects, those nothing else among them needs first,
// with everything that needs them. It leaves at least one suspect on, or returns nil when no split does (one package).
func split(mods []Mod, suspects []string) []string {
	needed := map[string]bool{}
	for _, m := range mods {
		if slices.Contains(suspects, m.ID) {
			for _, d := range m.Dependencies {
				needed[d] = true
			}
		}
	}
	order := slices.Clone(suspects)
	slices.SortStableFunc(order, func(a, b string) int {
		switch {
		case needed[a] == needed[b]:
			return 0
		case needed[a]:
			return 1
		}
		return -1
	})
	for k := (len(order) + 1) / 2; k >= 1; k-- {
		off := DisableClosure(mods, order[:k])
		if slices.ContainsFunc(suspects, func(id string) bool { return !slices.Contains(off, id) }) {
			return off
		}
	}
	return nil
}

func progressSteps(n int) int {
	steps := 0
	for n > 1 {
		steps++
		n = (n + 1) / 2
	}
	return steps
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
	slices.Sort(result)
	return result
}
