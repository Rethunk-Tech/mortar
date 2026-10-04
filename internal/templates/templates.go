// Package templates stores profile templates: a bundle of mods plus game settings and launch options that a new
// profile starts from.
package templates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/bundles"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/gamesettings"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

const maxName = 60

// Template is a named starting point for new profiles of one game.
type Template struct {
	Name          string                `json:"name"`
	Game          string                `json:"game"`
	Bundle        []bundles.Mod         `json:"bundle"`
	GameSettings  gamesettings.Settings `json:"gameSettings"`
	LaunchOptions string                `json:"launchOptions"`
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

func (s *Service) file(game string) (string, error) {
	if !gamepkg.Valid(game) {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", game))
	}
	return filepath.Join(s.root, game+".json"), nil
}

func (s *Service) read(game string) ([]Template, error) {
	path, err := s.file(game)
	if err != nil {
		return nil, err
	}
	list := []Template{}
	if _, err := datadir.ReadJSON(path, &list); err != nil {
		return nil, fmt.Errorf("read templates for %s: %w", game, err)
	}
	if list == nil {
		list = []Template{}
	}
	return list, nil
}

func (s *Service) write(game string, list []Template) error {
	path, err := s.file(game)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, list)
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

// SaveTemplateFromProfile captures a profile's mods, game settings and launch options under name, replacing a
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
	t := Template{Name: name, Game: game, Bundle: mods, GameSettings: settings, LaunchOptions: options}
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

// NewProfileFromTemplate creates a profile, applies the template's game settings and launch options, then adds the
// bundle's mods as a bundle apply does. Mods the store lacks come back in Missing for the caller to download. A
// failure after the profile exists moves it to the trash again.
func (s *Service) NewProfileFromTemplate(game, templateName, profileName string) (bundles.ApplyResult, error) {
	s.mu.Lock()
	list, err := s.read(game)
	s.mu.Unlock()
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	i := indexOf(list, templateName)
	if i < 0 {
		return bundles.ApplyResult{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("template %q was not found", templateName))
	}
	t := list[i]
	p, err := s.d.Profiles.Create(game, profileName)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	res, err := s.fill(game, p.ID, t)
	if err != nil {
		return bundles.ApplyResult{}, errors.Join(err, s.d.Profiles.Delete(game, p.ID))
	}
	return res, nil
}

func (s *Service) fill(game, id string, t Template) (bundles.ApplyResult, error) {
	if err := s.d.SetGameSettings(game, id, t.GameSettings); err != nil {
		return bundles.ApplyResult{}, err
	}
	if t.LaunchOptions != "" {
		if _, err := s.d.Profiles.SetLaunchOptions(game, id, t.LaunchOptions); err != nil {
			return bundles.ApplyResult{}, err
		}
	}
	return s.d.Bundles.ApplyMods(game, "template", t.Bundle, id)
}
