package profile

import (
	"fmt"
	"maps"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const overrideSkipPlayCheck = "skipPlayCheck"

// PrefOverrides is the map Resolve should see.
func (p Profile) PrefOverrides() map[string]string {
	if len(p.Overrides) == 0 {
		return nil
	}
	return maps.Clone(p.Overrides)
}

func errNotOverridable(key string) error {
	return fmt.Errorf("setting %q cannot be overridden on a profile", key)
}

// checkOverride rejects a setting a profile cannot override and a value the setting does not accept.
func checkOverride(game, key, value string) error {
	if !settings.ProfileOverridable(key) {
		return errNotOverridable(key)
	}
	tmp := settings.Settings{}
	return settings.ApplyKeyGame(&tmp, key, value, game)
}

// SetOverride writes one profile-overridable setting, or clears it when value is "default".
func (s *Store) SetOverride(game, id, key, value string) (Profile, error) {
	if value == "default" {
		if !settings.ProfileOverridable(key) {
			return Profile{}, errNotOverridable(key)
		}
	} else if err := checkOverride(game, key, value); err != nil {
		return Profile{}, err
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
		if err := checkOverride(game, key, value); err != nil {
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
