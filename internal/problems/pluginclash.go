package problems

import (
	"io/fs"
	"path/filepath"
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
// Keep is the key of the copy with the newest plugin version, the one "Keep newer" leaves enabled.
type PluginClash struct {
	GUID   string       `json:"guid"`
	Copies []PluginCopy `json:"copies"`
	Keep   string       `json:"keep"`
}

type pluginKey struct{ dir string }

var (
	pluginMu    sync.Mutex
	pluginCache = map[pluginKey][]dotnet.Plugin{}
)

// pluginsIn lists the plugins every DLL under dir declares. A store folder never changes for its key, so the answer is
// kept for the life of the process.
func pluginsIn(dir string) []dotnet.Plugin {
	pluginMu.Lock()
	cached, ok := pluginCache[pluginKey{dir}]
	pluginMu.Unlock()
	if ok {
		return cached
	}
	var out []dotnet.Plugin
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".dll") {
			return nil
		}
		plugins, _ := dotnet.Plugins(path)
		out = append(out, plugins...)
		return nil
	})
	pluginMu.Lock()
	pluginCache[pluginKey{dir}] = out
	pluginMu.Unlock()
	return out
}

// pluginClashes finds GUIDs shipped by two or more of the enabled packages, sorted by GUID.
func pluginClashes(pkgs []profile.PackageRef) []PluginClash {
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
			if display[g] == "" {
				display[g] = pl.GUID
			}
			byGUID[g] = append(byGUID[g], owner{p, pl.Version})
		}
	}
	var out []PluginClash
	for g, owners := range byGUID {
		if len(owners) < 2 {
			continue
		}
		c := PluginClash{GUID: display[g]}
		newest := 0
		for i, o := range owners {
			c.Copies = append(c.Copies, PluginCopy{Key: o.pkg.Key, ID: o.pkg.ID, Name: o.pkg.Name, Version: o.pkg.Version, PluginVersion: o.plugin})
			if cmp, ok := meta.CompareVersions(o.plugin, owners[newest].plugin); ok && cmp > 0 {
				newest = i
			}
		}
		c.Keep = owners[newest].pkg.Key
		slices.SortFunc(c.Copies, func(a, b PluginCopy) int { return strings.Compare(a.Key, b.Key) })
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b PluginClash) int { return strings.Compare(strings.ToLower(a.GUID), strings.ToLower(b.GUID)) })
	return out
}
