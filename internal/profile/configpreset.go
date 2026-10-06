package profile

import (
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
)

const historyConfigEdit = "config"

// ListConfigPresets names saved configs for a mod.
func (s *Service) ListConfigPresets(game string, uniqueID mod.ID) ([]string, error) {
	return modconfig.ListPresets(s.store.dataDir, game, uniqueID)
}

// SaveConfigPreset stores the given config.json text as a named preset.
func (s *Service) SaveConfigPreset(game string, uniqueID mod.ID, name, contents string) error {
	return modconfig.SavePreset(s.store.dataDir, game, uniqueID, name, []byte(contents))
}

// DeleteConfigPreset removes a named preset.
func (s *Service) DeleteConfigPreset(game string, uniqueID mod.ID, name string) error {
	return modconfig.DeletePreset(s.store.dataDir, game, uniqueID, name)
}

// ApplyConfigPreset writes a saved preset onto the profile's config.json and records one history event.
func (s *Service) ApplyConfigPreset(game, id, key string, uniqueID mod.ID, name string) error {
	body, err := modconfig.LoadPreset(s.store.dataDir, game, uniqueID, name)
	if err != nil {
		return err
	}
	return s.store.writeConfig(game, id, key, uniqueID, string(body), HistoryEvent{Change: ChangePresetApplied, Name: uniqueID.Local(), Detail: name})
}
