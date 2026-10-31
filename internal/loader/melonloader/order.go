package melonloader

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-bepinex/plugins"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Order is the order MelonLoader registers the enabled packages' melons in: every plugin, then every mod, each group
// sorted by MelonLoader's dependency walk and MelonPriority (dotnet.MelonOrder). A melon whose assembly is named by
// another's references, [MelonAdditionalDependencies] or [MelonOptionalDependencies] is that melon's dependency; one
// that is absent and not optional shows as missing.
func (Loader) Order(p loader.ProfileView) ([]loadorder.Row, error) {
	var plugs, mods []dotnet.Melon
	for _, dir := range p.Enabled {
		for _, path := range plugins.DLLs(dir) {
			m, err := dotnet.ScanMelon(path)
			if err != nil || !m.HasInfo {
				continue
			}
			rel, _ := filepath.Rel(dir, path)
			if slices.ContainsFunc(strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' }), func(s string) bool { return strings.EqualFold(s, "Plugins") }) {
				plugs = append(plugs, m)
			} else {
				mods = append(mods, m)
			}
		}
	}
	return melonRows(dotnet.MelonOrder(plugs), dotnet.MelonOrder(mods)), nil
}

func melonRows(groups ...[]dotnet.Melon) []loadorder.Row {
	var all []dotnet.Melon
	for _, g := range groups {
		all = append(all, g...)
	}
	id := func(assembly string) mod.ID { return mod.NewID(mod.FormatMelon, assembly) }
	present := map[string]bool{}
	for _, m := range all {
		present[m.Assembly] = true
	}
	dependents := map[string][]mod.ID{}
	rows := make([]loadorder.Row, 0, len(all))
	for i, m := range all {
		r := loadorder.Row{Position: i + 1, ID: id(m.Assembly), Name: m.Name, Cycle: m.Cycle}
		for _, name := range slices.Concat(m.Additional, m.Refs) {
			switch {
			case name == m.Assembly:
			case present[name]:
				if !slices.Contains(r.Required, id(name)) {
					r.Required = append(r.Required, id(name))
					dependents[name] = append(dependents[name], id(m.Assembly))
				}
			case slices.Contains(m.Additional, name) && !slices.Contains(m.Optional, name):
				r.Required = append(r.Required, id(name))
				r.MissingRequired = append(r.MissingRequired, id(name))
			}
		}
		for _, name := range m.Optional {
			if present[name] && !slices.Contains(r.Required, id(name)) {
				r.Optional = append(r.Optional, id(name))
			}
		}
		rows = append(rows, r)
	}
	for i := range rows {
		rows[i].Dependents = dependents[rows[i].ID.Local()]
	}
	return rows
}
