package profile

import (
	"fmt"
	"maps"
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

const (
	overrideUpdateBeforePlay = "updateModsBeforePlayDefault"
	overrideSkipPlayCheck    = "skipPlayCheck"
)

// PrefOverrides is the map Resolve should see, including folded legacy fields.
func (p Profile) PrefOverrides() map[string]string {
	out := map[string]string{}
	if p.Overrides != nil {
		out = maps.Clone(p.Overrides)
	}
	if _, ok := out[overrideUpdateBeforePlay]; !ok && p.UpdateBeforePlay {
		out[overrideUpdateBeforePlay] = "true"
	}
	if _, ok := out[overrideSkipPlayCheck]; !ok && p.SkipPlayCheck {
		out[overrideSkipPlayCheck] = "true"
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
		syncLegacyOverrides(p)
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
		syncLegacyOverrides(p)
		return nil
	})
}

func syncLegacyOverrides(p *Profile) {
	if v, ok := p.Overrides[overrideUpdateBeforePlay]; ok {
		on, err := strconv.ParseBool(v)
		p.UpdateBeforePlay = err == nil && on
	} else {
		p.UpdateBeforePlay = false
	}
	if v, ok := p.Overrides[overrideSkipPlayCheck]; ok {
		on, err := strconv.ParseBool(v)
		p.SkipPlayCheck = err == nil && on
	} else {
		p.SkipPlayCheck = false
	}
}
