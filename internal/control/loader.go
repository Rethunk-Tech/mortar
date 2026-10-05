package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// loaderOf checks that the game has a loader named want, or names the game's primary when want is empty.
func loaderOf(gameID, want string) (string, error) {
	l, ok := game.LoaderOf(gameID, want)
	if !ok {
		if want == "" {
			return "", fmt.Errorf("game %q has no loader", gameID)
		}
		return "", fmt.Errorf("game %q has no %q loader", gameID, want)
	}
	return l.ID(), nil
}

func (s *Services) loaderVersions(ctx context.Context, p Params) ([]string, error) {
	if s.Loaders == nil {
		return nil, fmt.Errorf("loader service is unavailable")
	}
	if _, err := loaderOf(p.Game, p.Loader); err != nil {
		return nil, err
	}
	return s.Loaders.ListVersions(ctx, p.Game, p.Loader)
}

func (s *Services) loaderInstall(ctx context.Context, p Params) (loader.Status, error) {
	if s.Loaders == nil {
		return loader.Status{}, fmt.Errorf("loader service is unavailable")
	}
	if _, err := loaderOf(p.Game, p.Loader); err != nil {
		return loader.Status{}, err
	}
	if p.Name == "" {
		return loader.Status{}, fmt.Errorf("loader install needs a version")
	}
	return s.Loaders.InstallVersion(ctx, p.Game, p.Loader, p.Name)
}

func (s *Services) loaderPin(p Params) error {
	if s.SettingsSvc == nil {
		return fmt.Errorf("settings are unavailable")
	}
	if p.Game == "" {
		return fmt.Errorf("loader pin needs a game")
	}
	id, err := loaderOf(p.Game, p.Loader)
	if err != nil {
		return err
	}
	key, ok := settings.PinKey(id)
	if !ok {
		return fmt.Errorf("the %s loader has no version pin", id)
	}
	return s.SettingsSvc.SetByKey(key, p.Value, p.Game)
}
