package problems

import (
	"cmp"
	"errors"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// PluginCopy is one enabled package that ships a plugin with a GUID another enabled package also ships.
type PluginCopy struct {
	Key     string `json:"key"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// PluginVersion is the version the plugin's [BepInPlugin] declares.
	PluginVersion string `json:"pluginVersion"`
}

// PluginClash is a [BepInPlugin] GUID declared by more than one enabled package; BepInEx loads only one of them.
// Keep is the key of the copy with the newest plugin version, the one "Keep newer" leaves enabled; between equal
// versions it is a copy whose package is not deprecated.
type PluginClash struct {
	GUID   string       `json:"guid"`
	Copies []PluginCopy `json:"copies"`
	Keep   string       `json:"keep"`
}

type pluginKey struct{ dir string }

var (
	pluginMu    sync.Mutex
	pluginCache = map[pluginKey]dotnet.Declared{}
)

// pluginsIn lists the plugins every DLL under dir declares.
func pluginsIn(dir string) []dotnet.Plugin { return declaredIn(dir).Plugins }

// declaredIn reads what every DLL under dir declares about its plugins. A store folder never changes for its key, so
// the answer is kept while the folder exists; a new folder usually comes with an update that removed an older one,
// so that is when the entries of removed folders go.
func declaredIn(dir string) dotnet.Declared {
	pluginMu.Lock()
	cached, ok := pluginCache[pluginKey{dir}]
	pluginMu.Unlock()
	if ok {
		return cached
	}
	out := dotnet.ScanDir(dir)
	pluginMu.Lock()
	maps.DeleteFunc(pluginCache, func(k pluginKey, _ dotnet.Declared) bool {
		_, err := os.Stat(k.dir)
		return errors.Is(err, fs.ErrNotExist)
	})
	pluginCache[pluginKey{dir}] = out
	pluginMu.Unlock()
	return out
}

// pluginClashes finds GUIDs shipped by two or more of the enabled packages, sorted by GUID. deprecated holds the keys
// of packages their author deprecated, developed those the player builds, whose copy is the one kept.
func pluginClashes(pkgs []profile.PackageRef, deprecated, developed map[string]bool) []PluginClash {
	type owner struct {
		pkg    profile.PackageRef
		plugin string
	}
	byGUID := map[string][]owner{}
	display := map[string]string{}
	for _, p := range pkgs {
		seen := map[string]bool{}
		for _, pl := range pluginsIn(p.Dir) {
			g := strings.ToLower(pl.GUID)
			if g == "" || seen[g] {
				continue
			}
			seen[g] = true
			display[g] = cmp.Or(display[g], pl.GUID)
			byGUID[g] = append(byGUID[g], owner{p, pl.Version})
		}
	}
	var out []PluginClash
	for g, owners := range byGUID {
		if len(owners) < 2 {
			continue
		}
		c := PluginClash{GUID: display[g]}
		for _, o := range owners {
			c.Copies = append(c.Copies, PluginCopy{Key: o.pkg.Key, ID: o.pkg.ID, Name: o.pkg.Name, Version: o.pkg.Version, PluginVersion: o.plugin})
		}
		pool := owners
		if own := slices.DeleteFunc(slices.Clone(owners), func(o owner) bool { return !developed[o.pkg.Key] }); len(own) > 0 {
			pool = own
		}
		newest := 0
		for i, o := range pool {
			cmp, ok := meta.CompareVersions(o.plugin, pool[newest].plugin)
			if ok && (cmp > 0 || cmp == 0 && deprecated[pool[newest].pkg.Key] && !deprecated[o.pkg.Key]) {
				newest = i
			}
		}
		c.Keep = pool[newest].pkg.Key
		slices.SortFunc(c.Copies, func(a, b PluginCopy) int { return strings.Compare(a.Key, b.Key) })
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b PluginClash) int { return strings.Compare(strings.ToLower(a.GUID), strings.ToLower(b.GUID)) })
	return out
}
