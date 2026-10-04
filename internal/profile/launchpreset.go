package profile

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/ids"
)

// BasePresetName is the display name of the profile's own launch settings, which are the preset every profile
// has and which named presets sit beside.
const BasePresetName = "Standard"

const maxLaunchPresets = 20

const showConsoleKey = "showSmapiConsole"

// LaunchPreset is one named launch configuration of a profile.
type LaunchPreset struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	LaunchOptions string `json:"launchOptions,omitempty"`
	LaunchPrefix  string `json:"launchPrefix,omitempty"`
	LaunchEnv     string `json:"launchEnv,omitempty"`
	// ShowConsole is "true" or "false" to override the SMAPI console setting, or empty to follow it.
	ShowConsole string `json:"showConsole,omitempty"`
}

// LaunchSpec is what one launch uses: the resolved preset's settings.
type LaunchSpec struct {
	Name        string
	Options     string
	Prefix      string
	Env         string
	ShowConsole string
}

// ResolvePreset picks the preset a launch uses. An empty name is the default preset; otherwise the name or id
// of a preset, or BasePresetName for the profile's own settings, matched ignoring case.
func (p Profile) ResolvePreset(name string) (LaunchSpec, error) {
	base := LaunchSpec{Name: BasePresetName, Options: p.LaunchOptions, Prefix: p.LaunchPrefix, Env: p.LaunchEnv}
	if name == "" {
		if p.DefaultLaunchPreset != "" {
			for _, preset := range p.LaunchPresets {
				if preset.ID == p.DefaultLaunchPreset {
					return preset.spec(), nil
				}
			}
		}
		return base, nil
	}
	if strings.EqualFold(name, BasePresetName) {
		return base, nil
	}
	for _, preset := range p.LaunchPresets {
		if preset.ID == name || strings.EqualFold(preset.Name, name) {
			return preset.spec(), nil
		}
	}
	return LaunchSpec{}, fmt.Errorf("profile %q has no launch preset %q", p.Name, name)
}

func (l LaunchPreset) spec() LaunchSpec {
	return LaunchSpec{Name: l.Name, Options: l.LaunchOptions, Prefix: l.LaunchPrefix, Env: l.LaunchEnv, ShowConsole: l.ShowConsole}
}

// Overrides returns the profile's setting overrides with the spec's console choice applied.
func (l LaunchSpec) Overrides(base map[string]string) map[string]string {
	if l.ShowConsole == "" {
		return base
	}
	out := maps.Clone(base)
	if out == nil {
		out = map[string]string{}
	}
	out[showConsoleKey] = l.ShowConsole
	return out
}

func validateLaunchPresets(presets []LaunchPreset, defaultID string) ([]LaunchPreset, error) {
	if len(presets) > maxLaunchPresets {
		return nil, fmt.Errorf("a profile holds at most %d launch presets", maxLaunchPresets)
	}
	out := make([]LaunchPreset, 0, len(presets))
	seenName := map[string]bool{strings.ToLower(BasePresetName): true}
	seenID := map[string]bool{}
	for _, preset := range presets {
		preset.Name = strings.TrimSpace(preset.Name)
		if preset.Name == "" {
			return nil, errors.New("a launch preset needs a name")
		}
		if key := strings.ToLower(preset.Name); seenName[key] {
			return nil, fmt.Errorf("launch preset name %q is already used", preset.Name)
		} else {
			seenName[key] = true
		}
		if preset.ID == "" || seenID[preset.ID] {
			preset.ID = ids.New()
		}
		seenID[preset.ID] = true
		if _, err := stardew.ParseLaunchOptions(preset.LaunchOptions); err != nil {
			return nil, fmt.Errorf("%s: %w", preset.Name, err)
		}
		if _, err := LaunchPrefixArgs(preset.LaunchPrefix); err != nil {
			return nil, fmt.Errorf("%s: %w", preset.Name, err)
		}
		if _, err := LaunchEnvironment(preset.LaunchEnv); err != nil {
			return nil, fmt.Errorf("%s: %w", preset.Name, err)
		}
		if preset.ShowConsole != "" && preset.ShowConsole != "true" && preset.ShowConsole != "false" {
			return nil, fmt.Errorf("%s: console must be true, false or empty", preset.Name)
		}
		out = append(out, preset)
	}
	if defaultID != "" && !seenID[defaultID] {
		return nil, errors.New("the default launch preset does not exist")
	}
	return out, nil
}

// SetLaunchPresets replaces a profile's named launch presets and which one is the default ("" is the profile's
// own settings). Presets without an id get one. It never touches mods/, so a running game does not block it.
func (s *Store) SetLaunchPresets(game, id string, presets []LaunchPreset, defaultID string) (Profile, error) {
	checked, err := validateLaunchPresets(presets, defaultID)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.LaunchPresets = checked
		p.DefaultLaunchPreset = defaultID
		return nil
	})
}

// AddLaunchPreset appends a copy of preset with a fresh id; a name already taken gets a number appended.
func (s *Store) AddLaunchPreset(game, id string, preset LaunchPreset) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		taken := map[string]bool{strings.ToLower(BasePresetName): true}
		for _, existing := range p.LaunchPresets {
			taken[strings.ToLower(existing.Name)] = true
		}
		base := strings.TrimSpace(preset.Name)
		preset.ID = ""
		preset.Name = base
		for n := 2; taken[strings.ToLower(preset.Name)]; n++ {
			preset.Name = fmt.Sprintf("%s %d", base, n)
		}
		checked, err := validateLaunchPresets(append(slices.Clone(p.LaunchPresets), preset), p.DefaultLaunchPreset)
		if err != nil {
			return err
		}
		p.LaunchPresets = checked
		return nil
	})
}

// SetDefaultLaunchPreset marks one preset as the one Play uses; "" selects the profile's own settings.
func (s *Store) SetDefaultLaunchPreset(game, id, presetID string) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		if presetID != "" && !containsPreset(p.LaunchPresets, presetID) {
			return errors.New("the default launch preset does not exist")
		}
		p.DefaultLaunchPreset = presetID
		return nil
	})
}

func containsPreset(presets []LaunchPreset, id string) bool {
	for _, preset := range presets {
		if preset.ID == id {
			return true
		}
	}
	return false
}

// LaunchSpec returns the settings a launch of the profile uses for the named preset.
func (s *Store) LaunchSpec(game, id, preset string) (LaunchSpec, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return LaunchSpec{}, err
	}
	return p.ResolvePreset(preset)
}
