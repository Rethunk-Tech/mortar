// Package gamesettings stores profile-specific Stardew Valley startup overrides.
package gamesettings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
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
	elementPattern = regexp.MustCompile(`(<([A-Za-z_][A-Za-z0-9_.:-]*)(?:\s[^>]*)?>)([^<]*)(</([A-Za-z_][A-Za-z0-9_.:-]*)>)`)
	readFile       = os.ReadFile
	writeFile      = datadir.WriteFile
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

func Patch(data []byte, settings Settings) ([]byte, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	values := map[string]string{}
	if settings.WindowMode != nil {
		values["windowMode"] = *settings.WindowMode
	}
	if settings.DisplayIndex != nil {
		values["displayIndex"] = strconv.Itoa(*settings.DisplayIndex)
	}
	if settings.PreferredResolutionX != nil {
		values["preferredResolutionX"] = strconv.Itoa(*settings.PreferredResolutionX)
	}
	if settings.PreferredResolutionY != nil {
		values["preferredResolutionY"] = strconv.Itoa(*settings.PreferredResolutionY)
	}
	if settings.FullscreenResolutionX != nil {
		values["fullscreenResolutionX"] = strconv.Itoa(*settings.FullscreenResolutionX)
	}
	if settings.FullscreenResolutionY != nil {
		values["fullscreenResolutionY"] = strconv.Itoa(*settings.FullscreenResolutionY)
	}
	if settings.ZoomLevel != nil {
		values["zoomLevel"] = strconv.Itoa(*settings.ZoomLevel)
	}
	if settings.UIScale != nil {
		values["uiScale"] = strconv.Itoa(*settings.UIScale)
	}
	if settings.StartMuted != nil {
		values["startMuted"] = strconv.FormatBool(*settings.StartMuted)
	}
	if settings.MusicVolumeLevel != nil {
		values["musicVolumeLevel"] = strconv.Itoa(*settings.MusicVolumeLevel)
	}
	if settings.SoundVolumeLevel != nil {
		values["soundVolumeLevel"] = strconv.Itoa(*settings.SoundVolumeLevel)
	}
	result := elementPattern.ReplaceAllFunc(data, func(element []byte) []byte {
		matches := elementPattern.FindSubmatch(element)
		if len(matches) == 0 {
			return element
		}
		name := string(matches[2])
		if name != string(matches[5]) {
			return element
		}
		value, ok := values[name]
		if !ok {
			return element
		}
		return bytes.Join([][]byte{matches[1], []byte(value), matches[4]}, nil)
	})
	return result, nil
}
