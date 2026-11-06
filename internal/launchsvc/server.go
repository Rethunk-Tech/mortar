package launchsvc

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/dotnetroot"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// serverMode reports whether the profile launches the game's dedicated server rather than the client.
func (s *Service) serverMode(gameID, profileID string) bool {
	return s.profiles != nil && profileID != "" && s.profiles.ServerMode(gameID, profileID)
}

// serverPlan turns the plan the folder loader made for the client into the start of the dedicated server: the same
// mods, placed in the server's data folder instead of the client's Mods, with the server in place of the client.
func (s *Service) serverPlan(plan *launchplan.Plan, gameID string, inst game.Install, profileID string) error {
	info, ok := components.Game(gameID)
	if !ok || info.ProfileServer == nil {
		return usererr.Wrap(usererr.Invalid, errors.New(info.Name+" has no dedicated server"))
	}
	if inst.Store == game.StoreFlatpak {
		return usererr.Wrap(usererr.Invalid, errors.New("the dedicated server does not run from a Flatpak install"))
	}
	dataDir, err := s.profiles.ServerDataDir(gameID, profileID)
	if err != nil {
		return err
	}
	exe, args, err := game.ProfileServerStart(*info.ProfileServer, inst.Platform, inst.Dir, dataDir)
	if err != nil {
		return err
	}
	if err := plan.OverrideExe("dedicated server", exe); err != nil {
		return err
	}
	plan.AddArgs(args...)
	if root := dotnetroot.FlatpakRoot(); root != "" {
		plan.SetEnv("DOTNET_ROOT", root)
	}
	plan.SetRootDir(folder.ModsRole, filepath.Join(dataDir, "Mods"))
	return nil
}

// serverModConfig is the folder the server reads its mods' settings from, for the swap that gives it the profile's.
func (s *Service) serverModConfig(gameID, profileID string) (string, error) {
	dataDir, err := s.profiles.ServerDataDir(gameID, profileID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "ModConfig"), nil
}
