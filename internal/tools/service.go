package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// Catalog runs the tools of a game's catalog entry, which need the launch pipeline: nil means the game's tools are the
// player's own only.
type Catalog interface {
	GameToolPresent(gameID, profileID string, t components.GameTool) bool
	RunGameTool(ctx context.Context, gameID, profileID string, t components.GameTool) error
}

// catalogPrefix starts the id of a catalog tool, which no tool the player adds can start with.
const catalogPrefix = "game:"

// Service exposes per-game external tools to the frontend.
type Service struct {
	store    *Store
	home     string
	settings *settings.Store
	profiles *profile.Store
	// Catalog is set once the launch service exists.
	Catalog Catalog
}

func NewService(home string, settings *settings.Store, profiles *profile.Store) (*Service, error) {
	store, err := Open()
	if err != nil {
		return nil, err
	}
	return &Service{store: store, home: home, settings: settings, profiles: profiles}, nil
}

// List is the catalog's tools of the game that are in the install profileID runs ("" for the game's own), then the
// player's own.
func (s *Service) List(game, profileID string) ([]Tool, error) {
	own, err := s.store.List(game)
	if err != nil {
		return nil, err
	}
	return append(s.catalogTools(game, profileID), own...), nil
}

func (s *Service) catalogTools(gameID, profileID string) []Tool {
	info, ok := components.Game(gameID)
	if !ok || s.Catalog == nil {
		return nil
	}
	var out []Tool
	for _, t := range info.Tools {
		if s.Catalog.GameToolPresent(gameID, profileID, t) {
			out = append(out, Tool{ID: catalogPrefix + t.ID, Name: t.Name, Executable: t.Exe, Arguments: t.Args, Builtin: true})
		}
	}
	return out
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
	if catalogID, ok := strings.CutPrefix(id, catalogPrefix); ok {
		return s.launchCatalog(game, profileID, catalogID)
	}
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
		return errToolNotFound()
	}
	ctx, err := s.context(game, profileID)
	if err != nil {
		return err
	}
	return launchTool(t, ctx)
}

func (s *Service) launchCatalog(gameID, profileID, id string) error {
	info, _ := components.Game(gameID)
	for _, t := range info.Tools {
		if t.ID == id && s.Catalog != nil {
			return s.Catalog.RunGameTool(context.Background(), gameID, profileID, t)
		}
	}
	return errToolNotFound()
}

func (s *Service) context(gameID, profileID string) (Context, error) {
	pin := s.profiles.InstallOf(gameID, profileID)
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), gameID, pin)
	if err != nil {
		return Context{}, err
	}
	if inst.Dir == "" {
		return Context{}, fmt.Errorf("%s is not installed", gameID)
	}
	install := absDir(inst.Dir)
	mods, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return Context{}, err
	}
	mods = absDir(mods)
	profileDir := absDir(filepath.Dir(mods))
	saves, err := s.savesDir(gameID, pin)
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

func (s *Service) savesDir(gameID, pin string) (string, error) {
	if !game.HasSaves(gameID) {
		return "", nil
	}
	dir, err := game.SavesDir(s.home, s.settings.Get(), gameID, pin)
	if err != nil {
		return "", err
	}
	return absDir(dir), nil
}
