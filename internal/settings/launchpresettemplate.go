package settings

import (
	"fmt"
	"strings"
)

// LaunchPresetTemplate is a named launch-options, prefix, and environment set for one game.
type LaunchPresetTemplate struct {
	Name    string `json:"name"`
	Options string `json:"options"`
	Prefix  string `json:"prefix"`
	Env     string `json:"env"`
	// ShowConsole is "true" or "false" to override the SMAPI console setting, or empty to follow it.
	ShowConsole string `json:"showConsole,omitempty"`
}

func checkTemplate(t LaunchPresetTemplate) (LaunchPresetTemplate, error) {
	name, err := presetName(t.Name)
	if err != nil {
		return t, err
	}
	t.Name = name
	if t.ShowConsole != "" && t.ShowConsole != "true" && t.ShowConsole != "false" {
		return t, fmt.Errorf("console must be true, false or empty")
	}
	return t, nil
}

func presetName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("preset name is required")
	}
	return trimmed, nil
}

func replacePreset(cur []LaunchPresetTemplate, preset LaunchPresetTemplate) []LaunchPresetTemplate {
	out := make([]LaunchPresetTemplate, 0, len(cur)+1)
	for _, item := range cur {
		if !strings.EqualFold(item.Name, preset.Name) {
			out = append(out, item)
		}
	}
	return append(out, preset)
}

func dropPreset(cur []LaunchPresetTemplate, name string) ([]LaunchPresetTemplate, error) {
	out := make([]LaunchPresetTemplate, 0, len(cur))
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

func writePresets(cur *Settings, game string, presets []LaunchPresetTemplate) {
	g := cur.GamePrefs(game)
	g.LaunchPresetTemplates = presets
	putGame(cur, game, g)
}

// ListLaunchPresetTemplates returns the launch presets stored for game.
func (s *Store) ListLaunchPresetTemplates(game string) []LaunchPresetTemplate {
	return append([]LaunchPresetTemplate(nil), s.Get().GamePrefs(game).LaunchPresetTemplates...)
}

// AddLaunchPresetTemplate stores preset for game, replacing an existing preset of the same name.
func (s *Store) AddLaunchPresetTemplate(game string, preset LaunchPresetTemplate) error {
	preset, err := checkTemplate(preset)
	if err != nil {
		return err
	}
	_, err = s.Update(func(cur *Settings) {
		writePresets(cur, game, replacePreset(cur.GamePrefs(game).LaunchPresetTemplates, preset))
	})
	return err
}

// RemoveLaunchPresetTemplate deletes the named preset for game.
func (s *Store) RemoveLaunchPresetTemplate(game, name string) error {
	trimmed, err := presetName(name)
	if err != nil {
		return err
	}
	next, err := dropPreset(s.Get().GamePrefs(game).LaunchPresetTemplates, trimmed)
	if err != nil {
		return err
	}
	_, err = s.Update(func(cur *Settings) {
		writePresets(cur, game, next)
	})
	return err
}
