package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// installedPlugin is one plugin a package ships.
type installedPlugin struct {
	pkg     profile.PackageRef
	version string
}

// pluginDeps checks the [BepInDependency] and [BepInIncompatibility] attributes of the enabled packages' plugins against
// every package's plugins, before a launch has to fail to say so. pkgs holds disabled packages too, so a library that is
// installed but switched off is named as that. A hard dependency that is absent, disabled or too old is a Missing row,
// an absent soft one an Optional row, and an incompatibility with another enabled package a LoadFailure.
func pluginDeps(pkgs []profile.PackageRef, mods []framework.Mod) ([]Missing, []LoadFailure) {
	declared := make(map[string]dotnet.Declared, len(pkgs))
	byGUID := map[string][]installedPlugin{}
	for _, p := range pkgs {
		d := declaredIn(p.Dir)
		declared[p.Key] = d
		for _, pl := range d.Plugins {
			g := strings.ToLower(pl.GUID)
			byGUID[g] = append(byGUID[g], installedPlugin{p, pl.Version})
		}
	}
	var missing []Missing
	var clashes []LoadFailure
	seen := map[string]bool{}
	for _, p := range pkgs {
		if !p.Enabled {
			continue
		}
		d := declared[p.Key]
		for _, r := range d.Relations {
			g := strings.ToLower(r.GUID)
			if g == "bepinex" || strings.HasPrefix(g, "bepinex.") || g == strings.ToLower(r.Plugin) {
				continue
			}
			name := pluginName(d, r.Plugin, p)
			if seen[p.Key+"\x00"+g+"\x00"+r.Kind] {
				continue
			}
			seen[p.Key+"\x00"+g+"\x00"+r.Kind] = true
			if r.Kind == dotnet.Incompatible {
				if other, ok := enabledProvider(byGUID[g], p.Key); ok {
					clashes = append(clashes, LoadFailure{
						Key: p.Key, ID: p.ID, Name: p.Name, Plugin: name, Kind: bepinex5.KindIncompatiblePlugin,
						Dependency: other.pkg.Name, Message: "incompatible with " + r.GUID,
					})
				}
				continue
			}
			if m, ok := unmetDependency(r, byGUID[g], mods, p, name); ok {
				missing = append(missing, m)
			}
		}
	}
	return missing, clashes
}

func pluginName(d dotnet.Declared, guid string, p profile.PackageRef) string {
	for _, pl := range d.Plugins {
		if pl.GUID == guid && pl.Name != "" {
			return pl.Name
		}
	}
	return p.Name
}

// enabledProvider is an enabled package other than self that ships one of the plugins.
func enabledProvider(providers []installedPlugin, self string) (installedPlugin, bool) {
	i := slices.IndexFunc(providers, func(x installedPlugin) bool { return x.pkg.Enabled && x.pkg.Key != self })
	if i < 0 {
		return installedPlugin{}, false
	}
	return providers[i], true
}

// unmetDependency is the row for a dependency the profile does not meet. A dependency with no plugin of that GUID but an
// enabled mod of that name (the same mod shipped under another GUID) is met, as an outside requirement is.
func unmetDependency(r dotnet.Relation, providers []installedPlugin, mods []framework.Mod, p profile.PackageRef, name string) (Missing, bool) {
	meets := func(x installedPlugin) bool {
		c, ok := meta.CompareVersions(x.version, r.MinVersion)
		return r.MinVersion == "" || !ok || c >= 0
	}
	soft := r.Kind == dotnet.SoftDependency
	if len(providers) == 0 {
		others := slices.DeleteFunc(slices.Clone(mods), func(x framework.Mod) bool { return x.Key == p.Key })
		if installedByName(others, r.GUID[strings.LastIndex(r.GUID, ".")+1:]) {
			return Missing{}, false
		}
	}
	row := Missing{DependentID: p.ID, DependentName: name, ID: mod.NewID(mod.FormatBepInEx, r.GUID), MinimumVersion: r.MinVersion, Reason: "absent", Optional: soft}
	enabled := slices.DeleteFunc(slices.Clone(providers), func(x installedPlugin) bool { return !x.pkg.Enabled })
	switch {
	case slices.ContainsFunc(enabled, meets):
		return Missing{}, false
	case soft && len(providers) > 0:
		return Missing{}, false
	case len(enabled) > 0:
		row.Reason, row.ID, row.InstalledVersion = "outdated", enabled[0].pkg.ID, highestPlugin(enabled)
	case len(providers) > 0:
		i := slices.IndexFunc(providers, meets)
		if i < 0 {
			row.Reason, row.ID, row.InstalledVersion = "outdated", providers[0].pkg.ID, highestPlugin(providers)
		} else {
			row.Reason, row.ID = "disabled", providers[i].pkg.ID
		}
	}
	return row, true
}

func highestPlugin(list []installedPlugin) string {
	best := list[0].version
	for _, x := range list[1:] {
		if c, ok := meta.CompareVersions(x.version, best); ok && c > 0 {
			best = x.version
		}
	}
	return best
}
