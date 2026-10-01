package launchsvc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/gamesettings"
)

const (
	gameSettingsFile       = "game-settings.json"
	startupPreferencesFile = "startup_preferences"
)

type settingsRestore struct {
	path     string
	original gamesettings.Settings
	written  gamesettings.Settings
	once     sync.Once
}

func (s *Service) GameSettings(game, id string) (gamesettings.Settings, error) {
	path, err := s.profileSettingsPath(game, id)
	if err != nil {
		return gamesettings.Settings{}, err
	}
	value, err := gamesettings.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return gamesettings.Settings{}, nil
	}
	return value, err
}

func (s *Service) SetGameSettings(game, id string, value gamesettings.Settings) error {
	path, err := s.profileSettingsPath(game, id)
	if err != nil {
		return err
	}
	return gamesettings.Save(path, value)
}

func (s *Service) profileSettingsPath(game, id string) (string, error) {
	modsDir, err := s.profiles.ModsDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(modsDir), gameSettingsFile), nil
}

func startupPreferencesPath() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "StardewValley", startupPreferencesFile), nil
}

func (s *Service) prepareGameSettings(game, id string) (*settingsRestore, bool, error) {
	value, err := s.GameSettings(game, id)
	if err != nil {
		return nil, false, err
	}
	if emptySettings(value) {
		return nil, false, nil
	}
	path, err := startupPreferencesPath()
	if err != nil {
		return nil, false, err
	}
	data, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	original, err := readStartupSettings(data, value)
	if err != nil {
		return nil, false, err
	}
	written := settingsPresent(value, original)
	if emptySettings(written) {
		return nil, false, nil
	}
	patched, err := gamesettings.Patch(data, written)
	if err != nil {
		return nil, false, err
	}
	if bytes.Equal(data, patched) {
		return nil, false, nil
	}
	if err := fsx.WriteFile(path, patched, 0o600); err != nil {
		return nil, false, err
	}
	return &settingsRestore{path: path, original: original, written: written}, false, nil
}

func (s *Service) restoreGameSettings(restore *settingsRestore) {
	if restore == nil {
		return
	}
	restore.once.Do(func() {
		data, err := fsx.ReadFile(restore.path)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			return
		}
		current, err := readStartupSettings(data, restore.written)
		if err != nil {
			return
		}
		original := settingsUnchanged(current, restore.written, restore.original)
		if emptySettings(original) {
			return
		}
		patched, err := gamesettings.Patch(data, original)
		if err != nil || bytes.Equal(data, patched) {
			return
		}
		_ = fsx.WriteFile(restore.path, patched, 0o600)
	})
}

func readStartupSettings(data []byte, wanted gamesettings.Settings) (gamesettings.Settings, error) {
	if emptySettings(wanted) {
		return gamesettings.Settings{}, nil
	}
	names := settingNames(wanted)
	var out gamesettings.Settings
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	startupDepth := -1
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return gamesettings.Settings{}, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			if startupDepth < 0 && element.Name.Local == "startup_preferences" {
				startupDepth = depth
			}
			if startupDepth >= 0 && depth == startupDepth+1 && names[element.Name.Local] {
				var value string
				if err := decoder.DecodeElement(&value, &element); err != nil {
					return gamesettings.Settings{}, err
				}
				if err := setSettingValue(&out, element.Name.Local, strings.TrimSpace(value)); err != nil {
					return gamesettings.Settings{}, err
				}
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if startupDepth >= 0 && depth == startupDepth {
				startupDepth = -1
			}
		}
	}
}

func setSettingValue(out *gamesettings.Settings, name, value string) error {
	switch name {
	case "windowMode":
		out.WindowMode = &value
	case "displayIndex":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.DisplayIndex = &parsed
	case "preferredResolutionX":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.PreferredResolutionX = &parsed
	case "preferredResolutionY":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.PreferredResolutionY = &parsed
	case "fullscreenResolutionX":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.FullscreenResolutionX = &parsed
	case "fullscreenResolutionY":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.FullscreenResolutionY = &parsed
	case "zoomLevel":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.ZoomLevel = &parsed
	case "uiScale":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.UIScale = &parsed
	case "startMuted":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.StartMuted = &parsed
	case "musicVolumeLevel":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.MusicVolumeLevel = &parsed
	case "soundVolumeLevel":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.SoundVolumeLevel = &parsed
	}
	return nil
}

func settingNames(value gamesettings.Settings) map[string]bool {
	return map[string]bool{
		"windowMode":            value.WindowMode != nil,
		"displayIndex":          value.DisplayIndex != nil,
		"preferredResolutionX":  value.PreferredResolutionX != nil,
		"preferredResolutionY":  value.PreferredResolutionY != nil,
		"fullscreenResolutionX": value.FullscreenResolutionX != nil,
		"fullscreenResolutionY": value.FullscreenResolutionY != nil,
		"zoomLevel":             value.ZoomLevel != nil,
		"uiScale":               value.UIScale != nil,
		"startMuted":            value.StartMuted != nil,
		"musicVolumeLevel":      value.MusicVolumeLevel != nil,
		"soundVolumeLevel":      value.SoundVolumeLevel != nil,
	}
}

func settingsPresent(value, present gamesettings.Settings) gamesettings.Settings {
	var out gamesettings.Settings
	if value.WindowMode != nil && present.WindowMode != nil {
		out.WindowMode = value.WindowMode
	}
	if value.DisplayIndex != nil && present.DisplayIndex != nil {
		out.DisplayIndex = value.DisplayIndex
	}
	if value.PreferredResolutionX != nil && present.PreferredResolutionX != nil {
		out.PreferredResolutionX = value.PreferredResolutionX
	}
	if value.PreferredResolutionY != nil && present.PreferredResolutionY != nil {
		out.PreferredResolutionY = value.PreferredResolutionY
	}
	if value.FullscreenResolutionX != nil && present.FullscreenResolutionX != nil {
		out.FullscreenResolutionX = value.FullscreenResolutionX
	}
	if value.FullscreenResolutionY != nil && present.FullscreenResolutionY != nil {
		out.FullscreenResolutionY = value.FullscreenResolutionY
	}
	if value.ZoomLevel != nil && present.ZoomLevel != nil {
		out.ZoomLevel = value.ZoomLevel
	}
	if value.UIScale != nil && present.UIScale != nil {
		out.UIScale = value.UIScale
	}
	if value.StartMuted != nil && present.StartMuted != nil {
		out.StartMuted = value.StartMuted
	}
	if value.MusicVolumeLevel != nil && present.MusicVolumeLevel != nil {
		out.MusicVolumeLevel = value.MusicVolumeLevel
	}
	if value.SoundVolumeLevel != nil && present.SoundVolumeLevel != nil {
		out.SoundVolumeLevel = value.SoundVolumeLevel
	}
	return out
}

func settingsUnchanged(current, written, original gamesettings.Settings) gamesettings.Settings {
	var out gamesettings.Settings
	if sameString(current.WindowMode, written.WindowMode) {
		out.WindowMode = original.WindowMode
	}
	if sameInt(current.DisplayIndex, written.DisplayIndex) {
		out.DisplayIndex = original.DisplayIndex
	}
	if sameInt(current.PreferredResolutionX, written.PreferredResolutionX) {
		out.PreferredResolutionX = original.PreferredResolutionX
	}
	if sameInt(current.PreferredResolutionY, written.PreferredResolutionY) {
		out.PreferredResolutionY = original.PreferredResolutionY
	}
	if sameInt(current.FullscreenResolutionX, written.FullscreenResolutionX) {
		out.FullscreenResolutionX = original.FullscreenResolutionX
	}
	if sameInt(current.FullscreenResolutionY, written.FullscreenResolutionY) {
		out.FullscreenResolutionY = original.FullscreenResolutionY
	}
	if sameInt(current.ZoomLevel, written.ZoomLevel) {
		out.ZoomLevel = original.ZoomLevel
	}
	if sameInt(current.UIScale, written.UIScale) {
		out.UIScale = original.UIScale
	}
	if sameBool(current.StartMuted, written.StartMuted) {
		out.StartMuted = original.StartMuted
	}
	if sameInt(current.MusicVolumeLevel, written.MusicVolumeLevel) {
		out.MusicVolumeLevel = original.MusicVolumeLevel
	}
	if sameInt(current.SoundVolumeLevel, written.SoundVolumeLevel) {
		out.SoundVolumeLevel = original.SoundVolumeLevel
	}
	return out
}

func sameString(a, b *string) bool {
	return a != nil && b != nil && *a == *b
}

func sameInt(a, b *int) bool {
	return a != nil && b != nil && *a == *b
}

func sameBool(a, b *bool) bool {
	return a != nil && b != nil && *a == *b
}

func emptySettings(value gamesettings.Settings) bool {
	return value.WindowMode == nil &&
		value.DisplayIndex == nil &&
		value.PreferredResolutionX == nil &&
		value.PreferredResolutionY == nil &&
		value.FullscreenResolutionX == nil &&
		value.FullscreenResolutionY == nil &&
		value.ZoomLevel == nil &&
		value.UIScale == nil &&
		value.StartMuted == nil &&
		value.MusicVolumeLevel == nil &&
		value.SoundVolumeLevel == nil
}
