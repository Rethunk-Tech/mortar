// Package templates stores profile templates: a bundle of mods plus game settings and launch options that a new
// profile starts from.
package templates

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/bundles"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/gamesettings"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const maxName = 60

// LaunchConfig is the per-profile launch state a template carries beyond game settings and launch options.
type LaunchConfig struct {
	LaunchPrefix        string                 `json:"launchPrefix"`
	LaunchEnv           string                 `json:"launchEnv"`
	Overrides           map[string]string      `json:"overrides"`
	LaunchPresets       []profile.LaunchPreset `json:"launchPresets"`
	DefaultLaunchPreset string                 `json:"defaultLaunchPreset"`
}

// Template is a named starting point for new profiles of one game. Disabled lists the ids of its mods that
// start switched off.
type Template struct {
	Name          string                `json:"name"`
	Game          string                `json:"game"`
	Bundle        []bundles.Mod         `json:"bundle"`
	Disabled      []mod.ID              `json:"disabled"`
	GameSettings  gamesettings.Settings `json:"gameSettings"`
	LaunchOptions string                `json:"launchOptions"`
	LaunchConfig
}

// Deps wires the service to the stores that hold the pieces a template captures.
type Deps struct {
	Profiles        *profile.Store
	Bundles         *bundles.Service
	GameSettings    func(game, profileID string) (gamesettings.Settings, error)
	SetGameSettings func(game, profileID string, s gamesettings.Settings) error
}

// Service exposes templates to the frontend.
type Service struct {
	d    Deps
	root string
	mu   sync.Mutex
}

// NewService opens the template store below the application's data directory.
func NewService(d Deps, dataDir string) *Service {
	return &Service{d: d, root: filepath.Join(dataDir, "templates")}
}

func (s *Service) read(game string) ([]Template, error) {
	path, err := gamepkg.File(s.root, game)
	if err != nil {
		return nil, err
	}
	list, err := readList(path)
	if err != nil {
		return nil, fmt.Errorf("read templates for %s: %w", game, err)
	}
	return list, nil
}

// templatesFile is the on-disk shape.
type templatesFile struct {
	FormatVersion int        `json:"formatVersion"`
	Templates     []Template `json:"templates"`
}

func readList(path string) ([]Template, error) {
	b, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Template{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f templatesFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if err := datadir.CheckVersion(f.FormatVersion); err != nil {
		return nil, err
	}
	list := f.Templates
	if list == nil {
		list = []Template{}
	}
	return list, nil
}

func (s *Service) write(game string, list []Template) error {
	path, err := gamepkg.File(s.root, game)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteVersioned(path, templatesFile{FormatVersion: datadir.FormatVersion, Templates: list})
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("template name is empty")
	}
	if utf8.RuneCountInString(name) > maxName {
		return "", fmt.Errorf("template name is longer than %d characters", maxName)
	}
	return name, nil
}

func indexOf(list []Template, name string) int {
	return slices.IndexFunc(list, func(t Template) bool { return strings.EqualFold(t.Name, name) })
}

// Templates lists the game's templates.
func (s *Service) Templates(game string) ([]Template, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(game)
}

// SaveTemplateFromProfile captures a profile's mods (with which are off), game settings and launch configuration under name, replacing a
// template of the same name.
func (s *Service) SaveTemplateFromProfile(game, profileID, name string) (Template, error) {
	name, err := cleanName(name)
	if err != nil {
		return Template{}, err
	}
	mods, err := s.d.Bundles.ProfileMods(game, profileID)
	if err != nil {
		return Template{}, err
	}
	settings, err := s.d.GameSettings(game, profileID)
	if err != nil {
		return Template{}, err
	}
	options, err := s.d.Profiles.LaunchOptions(game, profileID)
	if err != nil {
		return Template{}, err
	}
	p, err := s.profileOf(game, profileID)
	if err != nil {
		return Template{}, err
	}
	t := Template{Name: name, Game: game, Bundle: mods, Disabled: disabledMods(p), GameSettings: settings, LaunchOptions: options, LaunchConfig: launchOf(p)}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read(game)
	if err != nil {
		return Template{}, err
	}
	if i := indexOf(list, name); i >= 0 {
		list[i] = t
	} else {
		list = append(list, t)
	}
	return t, s.write(game, list)
}

// DeleteTemplate removes a template.
func (s *Service) DeleteTemplate(game, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.read(game)
	if err != nil {
		return err
	}
	i := indexOf(list, name)
	if i < 0 {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("template %q was not found", name))
	}
	return s.write(game, slices.Delete(list, i, i+1))
}

// NewProfileFromTemplate creates a profile and fills it from the template. Mods the store lacks come back in Missing
// for the caller to download. A failure after the profile exists moves it to the trash again.
func (s *Service) NewProfileFromTemplate(game, templateName, profileName string) (bundles.ApplyResult, error) {
	t, err := s.find(game, templateName)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	p, err := s.d.Profiles.Create(game, profileName)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	add, _, _ := split(p, t.Bundle)
	res, err := s.fill(game, p.ID, t, add)
	if err != nil {
		return bundles.ApplyResult{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
	}
	return res, nil
}

// ReferencedStoreKeys lists, per game, the store items still needed by saved templates.
func (s *Service) ReferencedStoreKeys() (map[string][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]string{}
	files, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		game := strings.TrimSuffix(file.Name(), ".json")
		if file.IsDir() || game == file.Name() || !gamepkg.Valid(game) {
			continue
		}
		list, err := s.read(game)
		if err != nil {
			return nil, err
		}
		for _, t := range list {
			for _, m := range t.Bundle {
				if m.EntryKey != "" {
					out[game] = append(out[game], m.EntryKey)
				}
			}
		}
	}
	return out, nil
}
