package profile

import (
	"path/filepath"

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
