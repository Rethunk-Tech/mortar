package thunderstore

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// bepInExPack is the loader package. Any 5.4.x request is satisfied by the newest 5.4.x, since mods name old pack
// versions that the newest one still serves.
const bepInExPack = "bepinex-bepinexpack"

func packageID(namespace, name string) string { return strings.ToLower(namespace + "-" + name) }

// versionParts reads "1.2.3" as numbers; a part that is not a number counts as 0.
func versionParts(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(s)
	}
	return out
}

func newer(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

// pick is the version of p that ref asks for: the newest when it names none, the newest 5.4.x for the loader pack,
// else exactly the named one.
func pick(p pkg, ref Ref) (string, error) {
	id := packageID(p.Owner, p.Name)
	if id == bepInExPack {
		if ref.Version != "" {
			if v := versionParts(ref.Version); v[0] != 5 || v[1] != 4 {
				return "", fmt.Errorf("%s asks for BepInExPack %s, but Mortar runs BepInEx 5.4.x", id, ref.Version)
			}
		}
		best := ""
		for _, v := range p.Versions {
			if vp := versionParts(v.Number); vp[0] == 5 && vp[1] == 4 && (best == "" || newer(v.Number, best)) {
				best = v.Number
			}
		}
		if best == "" {
			return "", fmt.Errorf("no BepInExPack 5.4.x in the index")
		}
		return best, nil
	}
	best := ""
	for _, v := range p.Versions {
		if v.Number == ref.Version {
			return v.Number, nil
		}
		if best == "" || newer(v.Number, best) {
			best = v.Number
		}
	}
	if ref.Version != "" || best == "" {
		return "", fmt.Errorf("%s %s is not in the index", id, ref.Version)
	}
	return best, nil
}

func parseDependency(s string) (Ref, error) {
	d, err := deps.Thunderstore(s)
	if err != nil {
		return Ref{}, err
	}
	namespace, name, _ := strings.Cut(strings.TrimPrefix(d.Target.Package, "thunderstore:"), "-")
	return Ref{Namespace: namespace, Name: name, Version: d.Constraint}, nil
}

// Closure resolves roots and everything they depend on from the community's cached index. Each package appears once,
// at the highest version anything in the closure asks for, and dependencies come before the packages that need them,
// in a deterministic order. A missing package, a loader other than BepInEx 5.4.x or a dependency cycle is an error
// naming the chain of requests that led to it.
func (d Driver) Closure(ctx context.Context, key string, roots []Ref, mortarVersion string) ([]Resolved, error) {
	list, err := d.packages(ctx, key, source.UserAgent(mortarVersion)+" (+https://mortar.rethunk.tech)")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]pkg, len(list))
	for _, p := range list {
		byID[packageID(p.Owner, p.Name)] = p
	}
	chosen := map[string]string{}
	// A later request can raise a version that an earlier visit used, so walk again until nothing rises; versions only
	// go up, so this ends.
	for {
		w := &walk{byID: byID, chosen: chosen, state: map[string]int{}}
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
						out = append(out, d.resolved(p, v))
					}
				}
			}
			return out, nil
		}
	}
}

type walk struct {
	byID   map[string]pkg
	chosen map[string]string
	state  map[string]int // 1 visiting, 2 done
	order  []string
	raised bool
}

func (w *walk) visit(ref Ref, chain []string) error {
	id := packageID(ref.Namespace, ref.Name)
	via := strings.Join(append(append([]string{}, chain...), id), " -> ")
	p, ok := w.byID[id]
	if !ok {
		return fmt.Errorf("%s is not in the index (requested by %s)", id, via)
	}
	v, err := pick(p, ref)
	if err != nil {
		return fmt.Errorf("%w (requested by %s)", err, via)
	}
	if cur := w.chosen[id]; cur == "" || newer(v, cur) {
		w.chosen[id] = v
		w.raised = w.raised || cur != ""
	}
	switch w.state[id] {
	case 1:
		return fmt.Errorf("dependency cycle: %s", via)
	case 2:
		return nil
	}
	w.state[id] = 1
	for _, ver := range p.Versions {
		if ver.Number != w.chosen[id] {
			continue
		}
		for _, dep := range ver.Deps {
			r, err := parseDependency(dep)
			if err != nil {
				return fmt.Errorf("%w (in %s)", err, via)
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
