package settings

import (
	"fmt"
	"strings"
)

// LaunchPreset is a named launch-options, prefix, and environment set for one game.
type LaunchPreset struct {
	Name    string `json:"name"`
	Options string `json:"options"`
	Prefix  string `json:"prefix"`
	Env     string `json:"env"`
}

func presetName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("preset name is required")
	}
	return trimmed, nil
}

func replacePreset(cur []LaunchPreset, preset LaunchPreset) []LaunchPreset {
	out := make([]LaunchPreset, 0, len(cur)+1)
	for _, item := range cur {
		if !strings.EqualFold(item.Name, preset.Name) {
			out = append(out, item)
		}
	}
	return append(out, preset)
}

func dropPreset(cur []LaunchPreset, name string) ([]LaunchPreset, error) {
	out := make([]LaunchPreset, 0, len(cur))
	found := false
	for _, item := range cur {
		if strings.EqualFold(item.Name, name) {
			found = true
			continue
		}
		out = append(out, item)
	}
	if !found {
		return nil, fmt.Errorf("preset %q not found", name)
	}
	return out, nil
}

func writePresets(cur *Settings, game string, presets []LaunchPreset) {
	g := cur.GamePrefs(game)
	g.LaunchPresets = presets
	putGame(cur, game, g)
}

// ListLaunchPresets returns the launch presets stored for game.
func (s *Store) ListLaunchPresets(game string) []LaunchPreset {
	return append([]LaunchPreset(nil), s.Get().GamePrefs(game).LaunchPresets...)
}

// AddLaunchPreset stores preset for game, replacing an existing preset of the same name.
func (s *Store) AddLaunchPreset(game string, preset LaunchPreset) error {
	name, err := presetName(preset.Name)
	if err != nil {
		return err
	}
	preset.Name = name
	_, err = s.Update(func(cur *Settings) {
		writePresets(cur, game, replacePreset(cur.GamePrefs(game).LaunchPresets, preset))
	})
	return err
}

// RemoveLaunchPreset deletes the named preset for game.
func (s *Store) RemoveLaunchPreset(game, name string) error {
	trimmed, err := presetName(name)
	if err != nil {
		return err
	}
	next, err := dropPreset(s.Get().GamePrefs(game).LaunchPresets, trimmed)
	if err != nil {
		return err
	}
	_, err = s.Update(func(cur *Settings) {
		writePresets(cur, game, next)
	})
	return err
}
