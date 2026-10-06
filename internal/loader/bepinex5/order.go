package bepinex5

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Order is the order BepInEx loads the enabled packages' plugins in. Where the profile's last LogOutput.log loaded
// some of them in another order, that order holds for them and their rows are marked LastRun.
func (l Loader) Order(p loader.ProfileView) ([]loadorder.Row, error) {
	var asms []dotnet.Declared
	for _, dir := range p.Enabled {
		asms = append(asms, dotnet.ScanEach(dir)...)
	}
	order := dotnet.LoadOrder(asms)
	path, _ := l.Path(p)
	raw, _ := fsx.ReadFile(path)
	moved := asLastRun(order, string(raw))
	return rows(order, moved), nil
}

// asLastRun reorders, in place, the plugins the log loaded into the order it loaded them, keeping every other plugin
// at its predicted place, and reports the GUIDs that moved.
func asLastRun(order []dotnet.Loaded, log string) map[string]bool {
	at := map[string]int{}
	n := 0
	for line := range strings.Lines(log) {
		m := loading.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if m == nil {
			continue
		}
		i := slices.IndexFunc(order, func(l dotnet.Loaded) bool { return names(l.Plugin, m[1]) })
		if i >= 0 {
			if _, seen := at[order[i].GUID]; !seen {
				at[order[i].GUID] = n
				n++
			}
		}
	}
	var slots []int
	var logged []dotnet.Loaded
	for i, l := range order {
		if _, ok := at[l.GUID]; ok {
			slots = append(slots, i)
			logged = append(logged, l)
		}
	}
	slices.SortStableFunc(logged, func(a, b dotnet.Loaded) int { return at[a.GUID] - at[b.GUID] })
	moved := map[string]bool{}
	for k, i := range slots {
		if order[i].GUID != logged[k].GUID {
			moved[logged[k].GUID] = true
		}
		order[i] = logged[k]
	}
	return moved
}

// names reports whether a Loading line's "Name Version" is the plugin, reading versions as System.Version does.
func names(p dotnet.Plugin, nameVersion string) bool {
	i := strings.LastIndexByte(nameVersion, ' ')
	if i < 0 || nameVersion[:i] != p.Name {
		return false
	}
	c, ok := meta.CompareVersions(nameVersion[i+1:], p.Version)
	return ok && c == 0 || nameVersion[i+1:] == p.Version
}

func rows(order []dotnet.Loaded, moved map[string]bool) []loadorder.Row {
	id := func(guid string) mod.ID { return mod.NewID(mod.FormatBepInEx, guid) }
	present := map[string]bool{}
	for _, l := range order {
		present[strings.ToLower(l.GUID)] = true
	}
	dependents := map[string][]mod.ID{}
	for _, l := range order {
		for _, d := range l.Deps {
			if present[strings.ToLower(d.GUID)] {
				dependents[strings.ToLower(d.GUID)] = append(dependents[strings.ToLower(d.GUID)], id(l.GUID))
			}
		}
	}
	out := make([]loadorder.Row, 0, len(order))
	for i, l := range order {
		r := loadorder.Row{Position: i + 1, ID: id(l.GUID), Name: l.Name, Dependents: dependents[strings.ToLower(l.GUID)], Cycle: l.Cycle, LastRun: moved[l.GUID]}
		for _, d := range l.Deps {
			here := present[strings.ToLower(d.GUID)]
			switch {
			case builtIn(d.GUID):
			// A soft dependency is an optional integration hook; one that is absent says nothing.
			case d.Kind == dotnet.SoftDependency:
				if here {
					r.Optional = append(r.Optional, id(d.GUID))
				}
			case here:
				r.Required = append(r.Required, id(d.GUID))
			default:
				r.Required = append(r.Required, id(d.GUID))
				r.MissingRequired = append(r.MissingRequired, id(d.GUID))
			}
		}
		out = append(out, r)
	}
	return out
}

// builtIn is a GUID BepInEx itself provides, which no package ships.
func builtIn(guid string) bool {
	g := strings.ToLower(guid)
	return g == "bepinex" || strings.HasPrefix(g, "bepinex.")
}
