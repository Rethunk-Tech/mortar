package settings

import "strings"

// ListLaunchPresetTemplates returns the launch presets stored for game.
func (s *Service) ListLaunchPresetTemplates(game string) []LaunchPresetTemplate {
	return s.store.ListLaunchPresetTemplates(game)
}

// FindLaunchPresetTemplate returns the named template for game, matched ignoring case.
func (s *Service) FindLaunchPresetTemplate(game, name string) (LaunchPresetTemplate, bool) {
	for _, t := range s.store.ListLaunchPresetTemplates(game) {
		if strings.EqualFold(t.Name, strings.TrimSpace(name)) {
			return t, true
		}
	}
	return LaunchPresetTemplate{}, false
}

// AddLaunchPresetTemplate stores a named launch preset for game, replacing the same name.
func (s *Service) AddLaunchPresetTemplate(game, name, options, prefix, env string) error {
	preset := LaunchPresetTemplate{Name: name, Options: options, Prefix: prefix, Env: env}
	trimmed, err := presetName(preset.Name)
	if err != nil {
		return err
	}
	preset.Name = trimmed
	return s.set(func(cur *Settings) {
		writePresets(cur, game, replacePreset(cur.GamePrefs(game).LaunchPresetTemplates, preset))
	})
}

// RemoveLaunchPresetTemplate deletes the named launch preset for game.
func (s *Service) RemoveLaunchPresetTemplate(game, name string) error {
	trimmed, err := presetName(name)
	if err != nil {
		return err
	}
	next, err := dropPreset(s.store.Get().GamePrefs(game).LaunchPresetTemplates, trimmed)
	if err != nil {
		return err
	}
	return s.set(func(cur *Settings) {
		writePresets(cur, game, next)
	})
}
