package control

import "fmt"

func (s *Services) gameLaunchPresets(p Params) (any, error) {
	if p.Game == "" {
		return nil, fmt.Errorf("launch-presets needs a game")
	}
	if s.SettingsSvc == nil {
		return nil, fmt.Errorf("settings are unavailable")
	}
	switch p.Sub {
	case "", "list":
		return s.SettingsSvc.ListLaunchPresets(p.Game), nil
	case "add":
		if err := s.SettingsSvc.AddLaunchPreset(p.Game, p.Name, p.Value, p.Path, p.Query); err != nil {
			return nil, err
		}
		return s.SettingsSvc.ListLaunchPresets(p.Game), nil
	case "remove":
		if err := s.SettingsSvc.RemoveLaunchPreset(p.Game, p.Name); err != nil {
			return nil, err
		}
		return s.SettingsSvc.ListLaunchPresets(p.Game), nil
	default:
		return nil, fmt.Errorf("unknown launch-presets command %q", p.Sub)
	}
}
