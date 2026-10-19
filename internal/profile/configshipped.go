package profile

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// ShippedConfig is the config.json the mod's package ships, which is the mod's defaults; ok is false when the
// package ships none or is no longer in the store.
func (s *Store) ShippedConfig(game, id, key string, uniqueID mod.ID) (string, bool) {
	p, err := s.read(game, id)
	if err != nil {
		return "", false
	}
	return s.shippedConfigOf(game, p, key, uniqueID)
}

func (s *Store) shippedConfigOf(game string, p Profile, key string, uniqueID mod.ID) (string, bool) {
	e, m, found := p.FindMod(key, uniqueID)
	if !found {
		return "", false
	}
	src, err := s.items.Path(game, e.StoreKey())
	if err != nil {
		return "", false
	}
	b, err := fsx.ReadFile(filepath.Join(src, filepath.FromSlash(m.Folder), configFile))
	if err != nil {
		return "", false
	}
	norm, err := rewriteConfigJSON(b)
	if err != nil {
		return "", false
	}
	return string(norm), true
}

// PluginGUIDs is the [BepInPlugin] GUIDs the package holding uniqueID ships, which name its BepInEx config files; nil
// for a mod with a folder of its own.
func (s *Store) PluginGUIDs(game, id string, uniqueID mod.ID) []string {
	p, err := s.read(game, id)
	if err != nil {
		return nil
	}
	return s.pluginGUIDsOf(game, p, uniqueID)
}

func (s *Store) pluginGUIDsOf(game string, p Profile, uniqueID mod.ID) []string {
	e, c, ok := p.FindMod("", uniqueID)
	if ok && c.ModID != "" {
		// A folder game's mod (Vintage Story) names its config files after the mod id its modinfo.json gives.
		return []string{c.ModID}
	}
	if !ok || !e.Package || s.items == nil {
		return nil
	}
	dir, err := s.items.Path(game, e.StoreKey())
	if err != nil {
		return nil
	}
	var out []string
	for _, pl := range dotnet.PluginsIn(dir) {
		out = append(out, pl.GUID)
	}
	return out
}

// InstalledPluginGUIDs is the [BepInPlugin] GUID of every plugin the profile holds: each package's, bundled ones
// included, and any DLL placed loose in its plugins folder. A config file named for one of them belongs to a live mod
// whether or not the game has run.
func (s *Store) InstalledPluginGUIDs(game, id string) []string {
	s.mu.Lock()
	p, dir, err := s.readDir(game, id)
	s.mu.Unlock()
	if err != nil {
		return nil
	}
	var out []string
	add := func(root string) {
		for _, pl := range dotnet.PluginsIn(root) {
			out = append(out, pl.GUID)
		}
	}
	if s.items != nil {
		for _, e := range p.Entries {
			if !e.Package {
				continue
			}
			if root, err := s.items.Path(game, e.StoreKey()); err == nil {
				add(root)
			}
		}
	}
	add(filepath.Join(dir, "BepInEx", "plugins"))
	return out
}
