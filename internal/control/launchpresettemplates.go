package control

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func (s *Services) gameLaunchPresetTemplates(p Params) (any, error) {
	if p.Game == "" {
		return nil, fmt.Errorf("launch-preset-templates needs a game")
	}
	if s.SettingsSvc == nil {
		return nil, fmt.Errorf("settings are unavailable")
	}
	switch p.Sub {
	case "", "list":
		return s.SettingsSvc.ListLaunchPresetTemplates(p.Game), nil
	case "add":
		if err := s.SettingsSvc.AddLaunchPresetTemplate(p.Game, p.Name, p.Value, p.Path, p.Query, p.Key); err != nil {
			return nil, err
		}
		return s.SettingsSvc.ListLaunchPresetTemplates(p.Game), nil
	case "use":
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		tmpl, ok := s.SettingsSvc.FindLaunchPresetTemplate(p.Game, p.Name)
		if !ok {
			return nil, fmt.Errorf("launch preset template %q not found", p.Name)
		}
		if _, err := s.Profiles.AddLaunchPreset(p.Game, prof.ID, profile.LaunchPreset{
			Name: tmpl.Name, LaunchOptions: tmpl.Options, LaunchPrefix: tmpl.Prefix, LaunchEnv: tmpl.Env, ShowConsole: tmpl.ShowConsole,
		}); err != nil {
			return nil, err
		}
		return s.SettingsSvc.ListLaunchPresetTemplates(p.Game), nil
	case "remove":
		if err := s.SettingsSvc.RemoveLaunchPresetTemplate(p.Game, p.Name); err != nil {
			return nil, err
		}
		return s.SettingsSvc.ListLaunchPresetTemplates(p.Game), nil
	default:
		return nil, fmt.Errorf("unknown launch-preset-templates command %q", p.Sub)
	}
}
