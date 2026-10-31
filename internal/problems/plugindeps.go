package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-bepinex/plugins"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"

	"github.com/Rethunk-Tech/go-semverish"
)

// installedPlugin is one plugin a package ships.
type installedPlugin struct {
	pkg     profile.PackageRef
	version string
}

// pluginDeps checks the [BepInDependency] and [BepInIncompatibility] attributes of the enabled packages' plugins against
// every package's plugins, before a launch has to fail to say so. pkgs holds disabled packages too, so a library that is
// installed but switched off is named as that. A hard dependency that is absent, disabled or too old is a Missing row
// and an incompatibility with another enabled package a LoadFailure. A soft dependency is an optional integration hook
// and says nothing.
func pluginDeps(pkgs []profile.PackageRef, mods []framework.Mod) ([]Missing, []LoadFailure) {
	declared := make(map[string]dotnet.Declared, len(pkgs))
	byGUID := map[string][]installedPlugin{}
	for _, p := range pkgs {
		d := declaredIn(p.Dir)
		declared[p.Key] = d
		for _, pl := range d.Plugins {
			if !plugins.Loadable(pl) {
				continue
			}
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
						Code: "incompatibleWith", Params: map[string]string{"list": r.GUID},
					})
				}
				continue
			}
			if r.Kind != dotnet.HardDependency {
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
	check := func(x installedPlugin) deps.Status {
		c, ok := semverish.CompareDotNet(x.version, r.MinVersion)
		switch {
		case r.MinVersion == "":
			return deps.Met
		case !ok:
			return deps.Unknown
		case c < 0:
			return deps.Unmet
		}
		return deps.Met
	}
	meets := func(x installedPlugin) bool { return check(x) == deps.Met }
	unordered := func(x installedPlugin) bool { return check(x) == deps.Unknown }
	if len(providers) == 0 {
		others := slices.DeleteFunc(slices.Clone(mods), func(x framework.Mod) bool { return x.Key == p.Key })
		if installedByName(others, r.GUID[strings.LastIndex(r.GUID, ".")+1:]) {
			return Missing{}, false
		}
	}
	row := Missing{DependentID: p.ID, DependentName: name, ID: mod.NewID(mod.FormatBepInEx, r.GUID), MinimumVersion: r.MinVersion, Reason: "absent"}
	enabled := slices.DeleteFunc(slices.Clone(providers), func(x installedPlugin) bool { return !x.pkg.Enabled })
	switch {
	case slices.ContainsFunc(enabled, meets):
		return Missing{}, false
	case slices.ContainsFunc(enabled, unordered):
		i := slices.IndexFunc(enabled, unordered)
		row.Reason, row.ID, row.InstalledVersion = "unknown", enabled[i].pkg.ID, enabled[i].version
	case len(enabled) > 0:
		row.Reason, row.ID, row.InstalledVersion = "outdated", enabled[0].pkg.ID, highestPlugin(enabled)
	case len(providers) > 0:
		i := slices.IndexFunc(providers, func(x installedPlugin) bool { return check(x) != deps.Unmet })
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
		if c, ok := semverish.CompareDotNet(x.version, best); ok && c > 0 {
			best = x.version
		}
	}
	return best
}

// dropCovered removes the rows a package's Thunderstore manifest already reports. The manifest names the dependency
// as a package, which is what Enable and Download act on, so it wins over the plugin GUID the assembly names: the two
// are one row when they name the same installed package, or when an absent package's name is the GUID's last part.
func dropCovered(rows, reported []Missing) []Missing {
	return slices.DeleteFunc(rows, func(d Missing) bool {
		guid := d.ID.Local()
		return slices.ContainsFunc(reported, func(e Missing) bool {
			if !mod.Equal(e.DependentID, d.DependentID) {
				return false
			}
			name := e.ID.Local()
			_, short, _ := strings.Cut(name, "-")
			last := guid[strings.LastIndex(guid, ".")+1:]
			return mod.Equal(e.ID, d.ID) || mod.LooseName(name) == mod.LooseName(guid) || mod.LooseName(short) == mod.LooseName(last)
		})
	})
}

// withPluginDependencies adds to each package's mod the packages its plugins' [BepInDependency] attributes name (hard as
// required, soft as optional) that its manifest does not list. A mod with no manifest (one built locally, say) names its
// libraries only there, and without them it would need nothing and its libraries would look unused.
func withPluginDependencies(mods []framework.Mod, pkgs []profile.PackageRef) {
	byGUID := map[string][]profile.PackageRef{}
	for _, p := range pkgs {
		for _, pl := range declaredIn(p.Dir).Plugins {
			g := strings.ToLower(pl.GUID)
			byGUID[g] = append(byGUID[g], p)
		}
	}
	for _, p := range pkgs {
		i := slices.IndexFunc(mods, func(m framework.Mod) bool { return m.Key == p.Key && mod.Equal(m.ModID(), p.ID) })
		if i < 0 {
			continue
		}
		for _, r := range declaredIn(p.Dir).Relations {
			if r.Kind == dotnet.Incompatible {
				continue
			}
			for _, dep := range byGUID[strings.ToLower(r.GUID)] {
				listed := slices.ContainsFunc(mods[i].Dependencies, func(d mod.Dependency) bool { return mod.Equal(d.ModID(), dep.ID) })
				if dep.Key == p.Key || listed {
					continue
				}
				// Clipped: the list may be a parsed manifest's, which the parse cache shares.
				mods[i].Dependencies = append(slices.Clip(mods[i].Dependencies),
					mod.Dependency{UniqueID: dep.ID.Local(), Format: dep.ID.Format(), Required: r.Kind == dotnet.HardDependency})
			}
		}
	}
}
