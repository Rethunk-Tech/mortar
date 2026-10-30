package thunderstore

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-semverish"
	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func packageID(namespace, name string) string { return strings.ToLower(namespace + "-" + name) }

// parseVersion reads a version; one that does not parse counts as 0.0.
func parseVersion(v string) semverish.Version {
	pv, _ := semverish.Parse(v)
	return pv
}

// pick is the version of p that ref asks for: the newest when it names none or, unless exactDeps, when it is a
// dependency's pin, which r2modman treats as a minimum; for a BepInEx loader pack the newest of the series Mortar runs it as (5.4.x, or the
// whole major the catalog names, 6 for BepInExPack_IL2CPP), since mods name old pack versions of the same major (5.3.1)
// that the newest one still serves; else exactly the named one.
func pick(p tstore.Package, ref tstore.Ref, minimum bool) (string, error) {
	id := p.ID()
	if components.IsBepInExPackage(p.Owner + "-" + p.Name) {
		major := components.LoaderPackageMajor(p.Owner + "-" + p.Name)
		series, inSeries := "5.4.x", func(v semverish.Version) bool { return v.Major == 5 && v.Minor == 4 }
		if major != 5 {
			series, inSeries = strconv.Itoa(major)+".x", func(v semverish.Version) bool { return v.Major == major }
		}
		if ref.Version != "" && parseVersion(ref.Version).Major != major {
			return "", fmt.Errorf("%s asks for %s %s, but Mortar runs BepInEx %s", id, p.Name, ref.Version, series)
		}
		best := ""
		for _, v := range p.Versions {
			if inSeries(parseVersion(v.Number)) && (best == "" || meta.Newer(v.Number, best)) {
				best = v.Number
			}
		}
		if best == "" {
			return "", fmt.Errorf("no %s %s in the index", p.Name, series)
		}
		return best, nil
	}
	best := ""
	for _, v := range p.Versions {
		if v.Number == ref.Version && !minimum {
			return v.Number, nil
		}
		if best == "" || meta.Newer(v.Number, best) {
			best = v.Number
		}
	}
	if best == "" || (ref.Version != "" && (!minimum || meta.Newer(ref.Version, best))) {
		return "", fmt.Errorf("%s %s is not in the index", id, ref.Version)
	}
	return best, nil
}

func parseDependency(s string) (tstore.Ref, error) {
	d, err := deps.Thunderstore(s)
	if err != nil {
		return tstore.Ref{}, err
	}
	namespace, name, _ := strings.Cut(mod.ID(d.Target.Package).Local(), "-")
	return tstore.Ref{Namespace: namespace, Name: name, Version: d.Constraint}, nil
}

// Deps is how Closure treats the roots' dependencies.
type Deps int

const (
	// DepsNewest installs each dependency at its newest version, as r2modman does.
	DepsNewest Deps = iota
	// DepsPinned installs each dependency at the version its author pinned, as Gale does.
	DepsPinned
	// DepsNone resolves the roots alone, each at its own version: a list installed exactly as it was shared.
	DepsNone
)

// Closure resolves roots and everything they depend on from the community's cached index. Each package appears once,
// at the highest version anything in the closure asks for, and dependencies come before the packages that need them,
// in a deterministic order; deps chooses the dependencies' versions, or leaves them out. A missing package, a loader
// pack outside the series Mortar runs it as or a dependency cycle is an error naming the chain of requests that led to it.
func (d Driver) Closure(ctx context.Context, key string, roots []tstore.Ref, mortarVersion string, deps Deps) ([]Resolved, error) {
	list, err := d.packages(ctx, key, mortarVersion)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]tstore.Package, len(list))
	for _, p := range list {
		byID[p.ID()] = p
	}
	// A root named at an exact version (an imported profile or modpack) keeps it when another root depends on it;
	// every other dependency takes the newest version, as r2modman installs them.
	exact := map[string]bool{}
	for _, r := range roots {
		if r.Version != "" {
			exact[packageID(r.Namespace, r.Name)] = true
		}
	}
	chosen := map[string]string{}
	// A later request can raise a version that an earlier visit used, so walk again until nothing rises; versions only
	// go up, so this ends.
	for {
		w := &walk{byID: byID, exact: exact, deps: deps, chosen: chosen, state: map[string]int{}, dropIns: d.DropIns, missing: map[string][]string{}}
		for _, r := range roots {
			if err := w.visit(r, nil); err != nil {
				return nil, err
			}
		}
		if !w.raised {
			out := make([]Resolved, 0, len(w.order))
			for _, id := range w.order {
				p := byID[id]
				for _, v := range p.Versions {
					if v.Number == chosen[id] {
						r := d.resolved(p, v)
						r.Missing = w.missing[id]
						out = append(out, r)
					}
				}
			}
			return out, nil
		}
	}
}

type walk struct {
	byID   map[string]tstore.Package
	exact  map[string]bool
	deps   Deps
	chosen map[string]string
	state  map[string]int // 1 visiting, 2 done
	order  []string
	raised bool
	// missing is, per package, the dependencies the index no longer lists; they are left out rather than failing the package.
	missing map[string][]string
	// dropIns swaps a dependency for its outright replacement; a root the user asked for stays as asked.
	dropIns map[string]string
}

// managers are other mod managers that communities list as packages. They are programs, not mods: nothing a profile
// holds could run them, so search leaves them out, a mod that lists one as a dependency installs without it, and
// asking for one by name is refused.
var managers = []string{"ebkr-r2modman", "kesomannen-galemodmanager"}

func isManager(owner, name string) bool {
	return slices.Contains(managers, packageID(owner, name))
}

func (w *walk) visit(ref tstore.Ref, chain []string) error {
	id := packageID(ref.Namespace, ref.Name)
	if isManager(ref.Namespace, ref.Name) {
		if chain == nil {
			return fmt.Errorf("%s is a mod manager, not a mod", id)
		}
		return nil
	}
	via := strings.Join(append(append([]string{}, chain...), id), " -> ")
	p, ok := w.byID[id]
	if !ok && chain != nil && components.IsBepInExPackage(ref.Namespace+"-"+ref.Name) {
		// A mod names the pack under the default namespace where the community lists its own; Mortar installs the loader itself.
		return nil
	}
	if !ok {
		return fmt.Errorf("%s is not in the index (requested by %s)", id, via)
	}
	v, err := pick(p, ref, chain != nil && !w.exact[id] && w.deps == DepsNewest)
	if err != nil {
		return fmt.Errorf("%w (requested by %s)", err, via)
	}
	if cur := w.chosen[id]; cur == "" || meta.Newer(v, cur) {
		w.chosen[id] = v
		w.raised = w.raised || cur != ""
	}
	if w.state[id] != 0 {
		// Packages that list each other (a mod and its compat pack) install together, in either order.
		return nil
	}
	w.state[id] = 1
	// A loader pack's own dependencies are other mods that depend back on it; Mortar installs the loader itself.
	loader := components.IsBepInExPackage(p.Owner + "-" + p.Name)
	for _, ver := range p.Versions {
		if ver.Number != w.chosen[id] || w.deps == DepsNone || loader {
			continue
		}
		for _, dep := range p.Dependencies(ver) {
			r, err := parseDependency(dep)
			if err != nil {
				return fmt.Errorf("%w (in %s)", err, via)
			}
			if to, ok := w.dropIns[packageID(r.Namespace, r.Name)]; ok {
				if ns, name, cut := strings.Cut(to, "-"); cut {
					r = tstore.Ref{Namespace: ns, Name: name}
				}
			}
			if dep := packageID(r.Namespace, r.Name); w.byID[dep].Name == "" && !isManager(r.Namespace, r.Name) &&
				!components.IsBepInExPackage(r.Namespace+"-"+r.Name) {
				w.missing[id] = append(w.missing[id], r.Namespace+"-"+r.Name)
				continue
			}
			if err := w.visit(r, append(chain, id+" "+ver.Number)); err != nil {
				return err
			}
		}
	}
	w.state[id] = 2
	w.order = append(w.order, id)
	return nil
}
