package tools

import (
	"fmt"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Service exposes per-game external tools to the frontend.
type Service struct {
	store    *Store
	home     string
	settings *settings.Store
	profiles *profile.Store
}

func NewService(home string, settings *settings.Store, profiles *profile.Store) (*Service, error) {
	store, err := Open()
	if err != nil {
		return nil, err
	}
	return &Service{store: store, home: home, settings: settings, profiles: profiles}, nil
}

func (s *Service) List(game string) ([]Tool, error) {
	return s.store.List(game)
}

func (s *Service) Add(game string, t Tool) (Tool, error) {
	t.Arguments = NormalizeArguments(t.Arguments)
	ctx, err := s.validationContext(game)
	if err != nil {
		return Tool{}, err
	}
	if err := validateTool(t, ctx); err != nil {
		return Tool{}, err
	}
	return s.store.Add(game, t)
}

func (s *Service) Update(game string, t Tool) error {
	t.Arguments = NormalizeArguments(t.Arguments)
	ctx, err := s.validationContext(game)
	if err != nil {
		return err
	}
	if err := validateTool(t, ctx); err != nil {
		return err
	}
	return s.store.Update(game, t)
}

func (s *Service) validationContext(gameID string) (Context, error) {
	install, err := game.InstallDir(s.home, s.settings.Get(), gameID)
	if err != nil {
		return Context{}, err
	}
	return Context{Game: absDir(install)}, nil
}

func (s *Service) Remove(game, id string) error {
	return s.store.Remove(game, id)
}

func (s *Service) Launch(game, profileID, id string) error {
	tools, err := s.store.load(game)
	if err != nil {
		return err
	}
	var t Tool
	for _, x := range tools {
		if x.ID == id {
			t = x
			break
		}
	}
	if t.ID == "" {
		return fmt.Errorf("tool not found")
	}
	ctx, err := s.context(game, profileID)
	if err != nil {
		return err
	}
	return launchTool(t, ctx)
}

func (s *Service) context(gameID, profileID string) (Context, error) {
	install, err := game.InstallDir(s.home, s.settings.Get(), gameID)
	if err != nil {
		return Context{}, err
	}
	if install == "" {
		return Context{}, fmt.Errorf("%s is not installed", gameID)
	}
	install = absDir(install)
	mods, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return Context{}, err
	}
	mods = absDir(mods)
	profileDir := absDir(filepath.Dir(mods))
	saves, err := s.savesDir(gameID)
	if err != nil {
		return Context{}, err
	}
	return Context{
		Game:    install,
		Mods:    mods,
		Saves:   saves,
		Profile: profileDir,
	}, nil
}

func (s *Service) savesDir(gameID string) (string, error) {
	if gameID != "stardew" {
		return "", nil
	}
	_, selected, _, err := game.Resolve(s.home, s.settings.Get(), gameID)
	if err != nil {
		return "", err
	}
	dir, err := game.SavesDir(selected, s.home)
	if err != nil {
		return "", err
	}
	return absDir(dir), nil
}
