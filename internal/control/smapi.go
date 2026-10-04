package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func (s *Services) smapiVersions(ctx context.Context, gameID string) ([]string, error) {
	if s.Loaders == nil {
		return nil, fmt.Errorf("loader service is unavailable")
	}
	return s.Loaders.ListVersions(ctx, gameID)
}

func (s *Services) smapiInstall(ctx context.Context, gameID, version string) (loader.Status, error) {
	if s.Loaders == nil {
		return loader.Status{}, fmt.Errorf("loader service is unavailable")
	}
	if version == "" {
		return loader.Status{}, fmt.Errorf("smapi install needs a version")
	}
	return s.Loaders.InstallVersion(ctx, gameID, version)
}

func (s *Services) smapiPin(gameID, version string) error {
	if s.SettingsSvc == nil {
		return fmt.Errorf("settings are unavailable")
	}
	if gameID == "" {
		return fmt.Errorf("smapi pin needs a game")
	}
	return s.SettingsSvc.SetByKey("smapiPin", version, gameID)
}
