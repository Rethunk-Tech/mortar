package control

import (
	"github.com/Rethunk-Tech/mortar/internal/configsvc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// configs edits the typed config files (BepInEx .cfg, Vintage Story ModConfig) the way the window's editor does.
func (s *Services) configs() *configsvc.Service {
	svc := &configsvc.Service{
		Profiles: s.Store,
		SetJSON: func(game, profileID string, uniqueID mod.ID, sets []profile.ConfigSet) error {
			return s.Profiles.SetConfigValues(game, profileID, "", uniqueID, sets)
		},
		Running:  s.Store.Running,
		Starting: s.Store.Running,
	}
	// Without the launch service a launch cannot be told from a running game, so both refuse edits.
	if s.Launches != nil {
		svc.Starting = s.Launches.Starting
	}
	return svc
}

// modConfigFields lists the mod's config.json fields followed by those of its typed config files.
func (s *Services) modConfigFields(cfg *configsvc.Service, game, id string, ref profile.EnableRef) ([]profile.ConfigField, error) {
	fields, fieldsErr := s.Profiles.ListConfigFields(game, id, ref.Key, ref.ID)
	typed, err := cfg.Fields(game, id, string(ref.ID))
	if err != nil {
		return nil, err
	}
	// A BepInEx or Vintage Story mod has no config.json to read, which is no error when typed files answer.
	if fieldsErr != nil && len(typed) == 0 {
		return nil, fieldsErr
	}
	return append(fields, typed...), nil
}
