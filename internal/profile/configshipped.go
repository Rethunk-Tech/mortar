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
	e, m, found := p.FindMod(key, uniqueID)
	if !found {
		return "", false
	}
	src, err := s.items.Path(game, e.Key)
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
	e, _, ok := p.FindMod("", uniqueID)
	if !ok || !e.Package || s.items == nil {
		return nil
	}
	dir, err := s.items.Path(game, e.Key)
	if err != nil {
		return nil
	}
	var out []string
	for _, pl := range dotnet.PluginsIn(dir) {
		out = append(out, pl.GUID)
	}
	return out
}
