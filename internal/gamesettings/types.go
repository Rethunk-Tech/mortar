// Package gamesettings stores profile-specific Stardew Valley startup overrides.
package gamesettings

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Rethunk-Tech/go-safefs"
)

type Settings struct {
	WindowMode            *string `json:"windowMode,omitempty"`
	DisplayIndex          *int    `json:"displayIndex,omitempty"`
	PreferredResolutionX  *int    `json:"preferredResolutionX,omitempty"`
	PreferredResolutionY  *int    `json:"preferredResolutionY,omitempty"`
	FullscreenResolutionX *int    `json:"fullscreenResolutionX,omitempty"`
	FullscreenResolutionY *int    `json:"fullscreenResolutionY,omitempty"`
	ZoomLevel             *int    `json:"zoomLevel,omitempty"`
	UIScale               *int    `json:"uiScale,omitempty"`
	StartMuted            *bool   `json:"startMuted,omitempty"`
	MusicVolumeLevel      *int    `json:"musicVolumeLevel,omitempty"`
	SoundVolumeLevel      *int    `json:"soundVolumeLevel,omitempty"`
}

var (
	readFile  = os.ReadFile
	writeFile = safefs.WriteFileAtomic
)

func (s Settings) Validate() error {
	if s.WindowMode != nil {
		switch *s.WindowMode {
		case "windowed", "fullscreen", "borderless":
		default:
			return fmt.Errorf("invalid window mode %q", *s.WindowMode)
		}
	}
	for name, value := range map[string]*int{
		"displayIndex": s.DisplayIndex, "preferredResolutionX": s.PreferredResolutionX,
		"preferredResolutionY": s.PreferredResolutionY, "fullscreenResolutionX": s.FullscreenResolutionX,
		"fullscreenResolutionY": s.FullscreenResolutionY,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	for name, value := range map[string]*int{
		"zoomLevel": s.ZoomLevel, "uiScale": s.UIScale,
		"musicVolumeLevel": s.MusicVolumeLevel, "soundVolumeLevel": s.SoundVolumeLevel,
	} {
		if value != nil && (*value < 0 || *value > 100) {
			return fmt.Errorf("%s must be between 0 and 100", name)
		}
	}
	return nil
}

func Load(path string) (Settings, error) {
	data, err := readFile(path)
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func Save(path string, settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(path, data, 0o600)
}
