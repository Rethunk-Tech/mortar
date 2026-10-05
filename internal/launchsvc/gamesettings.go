package launchsvc

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/gamesettings"
)

const (
	gameSettingsFile    = "game-settings.json"
	settingsRestoreFile = "game-settings.restore.json"
)

type settingsRestore struct {
	path       string
	recordPath string
	original   gamesettings.Settings
	written    gamesettings.Settings
	once       sync.Once
}

type settingsRestoreRecord struct {
	Path     string                `json:"path"`
	Original gamesettings.Settings `json:"original"`
	Written  gamesettings.Settings `json:"written"`
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

func (s *Service) prepareGameSettings(gameID, id, installID string) (*settingsRestore, bool, error) {
	value, err := s.GameSettings(gameID, id)
	if err != nil {
		return nil, false, err
	}
	if !game.HasStartupSettings(gameID) || emptySettings(value) {
		return nil, false, nil
	}
	path, err := game.StartupPreferencesPath(s.home, s.settings.Get(), gameID, s.pinOf(gameID, id, installID))
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
	settingsPath, err := s.profileSettingsPath(gameID, id)
	if err != nil {
		return nil, false, err
	}
	recordPath := filepath.Join(filepath.Dir(settingsPath), settingsRestoreFile)
	restore := &settingsRestore{
		path: path, recordPath: recordPath, original: original, written: written,
	}
	if err := writeSettingsRestore(recordPath, restore); err != nil {
		return nil, false, err
	}
	if err := datadir.WriteFile(path, patched, 0o600); err != nil {
		return nil, false, errors.Join(err, os.Remove(recordPath))
	}
	return restore, false, nil
}

func writeSettingsRestore(path string, restore *settingsRestore) error {
	return datadir.WriteJSON(path, settingsRestoreRecord{Path: restore.path, Original: restore.original, Written: restore.written})
}

func readSettingsRestore(path string) (*settingsRestore, error) {
	body, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record settingsRestoreRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, err
	}
	return &settingsRestore{
		path: record.Path, recordPath: path, original: record.Original, written: record.Written,
	}, nil
}

func removeSettingsRestore(path string) error {
	if err := os.Remove(path); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Service) restoreGameSettings(restore *settingsRestore) error {
	if restore == nil {
		return nil
	}
	var restoreErr error
	restore.once.Do(func() {
		data, err := fsx.ReadFile(restore.path)
		if errors.Is(err, os.ErrNotExist) {
			restoreErr = err
			return
		}
		if err != nil {
			restoreErr = err
			return
		}
		current, err := readStartupSettings(data, restore.written)
		if err != nil {
			restoreErr = err
			return
		}
		original := settingsUnchanged(current, restore.written, restore.original)
		if emptySettings(original) {
			restoreErr = removeSettingsRestore(restore.recordPath)
			return
		}
		patched, err := gamesettings.Patch(data, original)
		if err != nil {
			restoreErr = err
			return
		}
		if bytes.Equal(data, patched) {
			restoreErr = removeSettingsRestore(restore.recordPath)
			return
		}
		if err := datadir.WriteFile(restore.path, patched, 0o600); err != nil {
			restoreErr = err
			return
		}
		restoreErr = removeSettingsRestore(restore.recordPath)
	})
	return restoreErr
}

// RecoverGameSettings applies records left by a launch that ended before its in-memory restore ran.
//
//wails:ignore
func (s *Service) RecoverGameSettings() error {
	var errs []error
	for _, id := range game.Implemented() {
		if game.HasStartupSettings(id) {
			errs = append(errs, s.recoverGameSettings(id))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) recoverGameSettings(gameID string) error {
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range profiles {
		if p.Error != "" {
			continue
		}
		settingsPath, err := s.profileSettingsPath(gameID, p.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		recordPath := filepath.Join(filepath.Dir(settingsPath), settingsRestoreFile)
		restore, err := readSettingsRestore(recordPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		if err := s.restoreGameSettings(restore); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
		}
	}
	return errors.Join(errs...)
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
