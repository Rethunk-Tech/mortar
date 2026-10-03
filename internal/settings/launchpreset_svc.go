package settings

// ListLaunchPresets returns the launch presets stored for game.
func (s *Service) ListLaunchPresets(game string) []LaunchPreset {
	return s.store.ListLaunchPresets(game)
}

// AddLaunchPreset stores a named launch preset for game, replacing the same name.
func (s *Service) AddLaunchPreset(game, name, options, prefix, env string) error {
	preset := LaunchPreset{Name: name, Options: options, Prefix: prefix, Env: env}
	trimmed, err := presetName(preset.Name)
	if err != nil {
		return err
	}
	preset.Name = trimmed
	return s.set(func(cur *Settings) {
		writePresets(cur, game, replacePreset(cur.GamePrefs(game).LaunchPresets, preset))
	})
}

// RemoveLaunchPreset deletes the named launch preset for game.
func (s *Service) RemoveLaunchPreset(game, name string) error {
	trimmed, err := presetName(name)
	if err != nil {
		return err
	}
	next, err := dropPreset(s.store.Get().GamePrefs(game).LaunchPresets, trimmed)
	if err != nil {
		return err
	}
	return s.set(func(cur *Settings) {
		writePresets(cur, game, next)
	})
}
