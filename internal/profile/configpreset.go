package profile

import (
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
)

const historyConfigPreset = "config"

// ListConfigPresets names saved configs for a mod.
func (s *Service) ListConfigPresets(game, uniqueID string) ([]string, error) {
	root, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	return modconfig.ListPresets(root, game, uniqueID)
}

// SaveConfigPreset stores the given config.json text as a named preset.
func (s *Service) SaveConfigPreset(game, uniqueID, name, contents string) error {
	root, err := datadir.Dir()
	if err != nil {
		return err
	}
	return modconfig.SavePreset(root, game, uniqueID, name, []byte(contents))
}

// DeleteConfigPreset removes a named preset.
func (s *Service) DeleteConfigPreset(game, uniqueID, name string) error {
	root, err := datadir.Dir()
	if err != nil {
		return err
	}
	return modconfig.DeletePreset(root, game, uniqueID, name)
}

// ApplyConfigPreset writes a saved preset onto the profile's config.json and records one history event.
func (s *Service) ApplyConfigPreset(game, id, key, uniqueID, name string) error {
	root, err := datadir.Dir()
	if err != nil {
		return err
	}
	body, err := modconfig.LoadPreset(root, game, uniqueID, name)
	if err != nil {
		return err
	}
	if err := s.store.WriteConfig(game, id, key, uniqueID, string(body)); err != nil {
		return err
	}
	return s.store.recordConfigPreset(game, id, "Applied preset "+name)
}

func (s *Store) recordConfigPreset(game, id, label string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.historyKind = historyConfigPreset
	s.historyLabel = label
	_, err := s.updateLocked(game, id, func(*Profile, string) error { return nil })
	return err
}
