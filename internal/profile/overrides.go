package profile

import (
	"fmt"
	"maps"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

const overrideSkipPlayCheck = "skipPlayCheck"

// PrefOverrides is the map Resolve should see.
func (p Profile) PrefOverrides() map[string]string {
	if len(p.Overrides) == 0 {
		return nil
	}
	return maps.Clone(p.Overrides)
}

// SetOverride writes one profile-overridable setting, or clears it when value is "default".
func (s *Store) SetOverride(game, id, key, value string) (Profile, error) {
	if !settings.ProfileOverridable(key) {
		return Profile{}, fmt.Errorf("setting %q cannot be overridden on a profile", key)
	}
	if value != "default" {
		tmp := settings.Settings{}
		if err := settings.ApplyKeyGame(&tmp, key, value, game); err != nil {
			return Profile{}, err
		}
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		if p.Overrides == nil {
			p.Overrides = map[string]string{}
		}
		if value == "default" {
			delete(p.Overrides, key)
		} else {
			p.Overrides[key] = value
		}
		if len(p.Overrides) == 0 {
			p.Overrides = nil
		}
		return nil
	})
}

// SetOverrides replaces the profile override map.
func (s *Store) SetOverrides(game, id string, overrides map[string]string) (Profile, error) {
	if overrides == nil {
		overrides = map[string]string{}
	}
	for key, value := range overrides {
		if !settings.ProfileOverridable(key) {
			return Profile{}, fmt.Errorf("setting %q cannot be overridden on a profile", key)
		}
		tmp := settings.Settings{}
		if err := settings.ApplyKeyGame(&tmp, key, value, game); err != nil {
			return Profile{}, err
		}
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		if len(overrides) == 0 {
			p.Overrides = nil
		} else {
			p.Overrides = maps.Clone(overrides)
		}
		return nil
	})
}
