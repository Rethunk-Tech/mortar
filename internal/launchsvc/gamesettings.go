package launchsvc

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
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
	patched, err := patchStartupSettings(data, written)
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
		return nil, false, errors.Join(err, fsx.Remove(recordPath))
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
	if err := fsx.Remove(path); !errors.Is(err, os.ErrNotExist) {
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
		patched, err := patchStartupSettings(data, original)
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

const (
	startupRoot    = "StartupPreferences"
	startupOptions = "clientOptions"
)

// startupSection is where the game keeps each managed element under the root.
var startupSection = map[string]string{
	"windowMode": "", "displayIndex": "", "fullscreenResolutionX": "", "fullscreenResolutionY": "", "startMuted": "",
	"preferredResolutionX": startupOptions, "preferredResolutionY": startupOptions,
	"zoomLevel": startupOptions, "uiScale": startupOptions,
	"musicVolumeLevel": startupOptions, "soundVolumeLevel": startupOptions,
	"fullscreen": startupOptions, "windowedBorderlessFullscreen": startupOptions,
}

// The game writes these as fractions (0.75); Mortar's settings hold percent.
var startupPercent = map[string]bool{"zoomLevel": true, "uiScale": true, "musicVolumeLevel": true, "soundVolumeLevel": true}

// The game's windowMode enum, read from a file it wrote while borderless.
var (
	windowModeCodes = map[string]string{"windowed": "0", "borderless": "1", "fullscreen": "2"}
	windowModeNames = map[string]string{"0": "windowed", "1": "borderless", "2": "fullscreen"}
)

// The game also keeps the window mode as these two flags, in the same order as windowModeCodes.
var windowModeFlags = map[string][2]string{
	"windowed": {"false", "false"}, "borderless": {"false", "true"}, "fullscreen": {"true", "false"},
}

type startupElement struct {
	name       string
	start, end int // byte range of the element's text
	text       string
}

// scanStartup lists the managed elements of data at their section paths, in document order.
func scanStartup(data []byte, names map[string]bool) ([]startupElement, error) {
	var out []startupElement
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var path []string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			name := element.Name.Local
			path = append(path, name)
			if !names[name] || len(path) < 2 || path[0] != startupRoot ||
				strings.Join(path[1:len(path)-1], "/") != startupSection[name] {
				continue
			}
			start := int(decoder.InputOffset())
			if err := decoder.Skip(); err != nil {
				return nil, err
			}
			path = path[:len(path)-1]
			end := bytes.LastIndexByte(data[:decoder.InputOffset()], '<')
			if end >= start {
				out = append(out, startupElement{name: name, start: start, end: end, text: strings.TrimSpace(string(data[start:end]))})
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}
}

func readStartupSettings(data []byte, wanted gamesettings.Settings) (gamesettings.Settings, error) {
	if emptySettings(wanted) {
		return gamesettings.Settings{}, nil
	}
	elements, err := scanStartup(data, settingNames(wanted))
	if err != nil {
		return gamesettings.Settings{}, err
	}
	var out gamesettings.Settings
	for _, element := range elements {
		if err := setSettingValue(&out, element.name, element.text); err != nil {
			return gamesettings.Settings{}, err
		}
	}
	return out, nil
}

// patchStartupSettings replaces the text of each set element and leaves every other byte alone.
func patchStartupSettings(data []byte, value gamesettings.Settings) ([]byte, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	names := settingNames(value)
	if value.WindowMode != nil {
		names["fullscreen"], names["windowedBorderlessFullscreen"] = true, true
	}
	elements, err := scanStartup(data, names)
	if err != nil {
		return nil, err
	}
	values := startupValues(value)
	out := append([]byte(nil), data...)
	for i := len(elements) - 1; i >= 0; i-- {
		e := elements[i]
		out = append(out[:e.start], append([]byte(values[e.name]), out[e.end:]...)...)
	}
	return out, nil
}

func intSlot(s *gamesettings.Settings, name string) **int {
	switch name {
	case "displayIndex":
		return &s.DisplayIndex
	case "preferredResolutionX":
		return &s.PreferredResolutionX
	case "preferredResolutionY":
		return &s.PreferredResolutionY
	case "fullscreenResolutionX":
		return &s.FullscreenResolutionX
	case "fullscreenResolutionY":
		return &s.FullscreenResolutionY
	case "zoomLevel":
		return &s.ZoomLevel
	case "uiScale":
		return &s.UIScale
	case "musicVolumeLevel":
		return &s.MusicVolumeLevel
	case "soundVolumeLevel":
		return &s.SoundVolumeLevel
	}
	return nil
}

func startupValues(value gamesettings.Settings) map[string]string {
	out := map[string]string{}
	if value.WindowMode != nil {
		out["windowMode"] = windowModeCodes[*value.WindowMode]
		flags := windowModeFlags[*value.WindowMode]
		out["fullscreen"], out["windowedBorderlessFullscreen"] = flags[0], flags[1]
	}
	if value.StartMuted != nil {
		out["startMuted"] = strconv.FormatBool(*value.StartMuted)
	}
	for name := range startupSection {
		slot := intSlot(&value, name)
		if slot == nil || *slot == nil {
			continue
		}
		if startupPercent[name] {
			out[name] = strconv.FormatFloat(float64(**slot)/100, 'f', -1, 64)
		} else {
			out[name] = strconv.Itoa(**slot)
		}
	}
	return out
}

func setSettingValue(out *gamesettings.Settings, name, value string) error {
	switch name {
	case "windowMode":
		if mode, ok := windowModeNames[value]; ok {
			out.WindowMode = &mode
		}
		return nil
	case "startMuted":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		out.StartMuted = &parsed
		return nil
	}
	slot := intSlot(out, name)
	if slot == nil {
		return nil
	}
	var parsed int
	if startupPercent[name] {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		parsed = int(math.Round(f * 100))
	} else {
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		parsed = n
	}
	*slot = &parsed
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
