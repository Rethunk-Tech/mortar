package profile

import (
	"context"
	"errors"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

func (s *Service) extraModsFolders(game string) []string {
	return s.settings.Get().GamePrefs(game).ExtraModsFolderList()
}

// ExtraFolderMods lists the mods in the game's extra mods folders (setting extraModsFolder) as Import from the game's
// Mods folder previews them, each mod's Folder inside the folder it came from; none set lists none.
func (s *Service) ExtraFolderMods(game string) (GameModsPreview, error) {
	out := GameModsPreview{Mods: []GameModPreview{}}
	for _, dir := range s.extraModsFolders(game) {
		got, err := s.store.PreviewGameMods(game, dir)
		if err != nil {
			return GameModsPreview{}, err
		}
		out.Mods = append(out.Mods, got.Mods...)
		out.CanMove = out.CanMove || got.CanMove
	}
	return out, nil
}

// InstallExtraFolderMod installs folder, a top-level folder of one extra mods folder as ExtraFolderMods returned it,
// into the profile the way a dropped folder is installed.
func (s *Service) InstallExtraFolderMod(ctx context.Context, game, id, folder string) (InstallResult, error) {
	roots := s.extraModsFolders(game)
	if len(roots) == 0 {
		return InstallResult{}, errors.New("no extra mods folder is set")
	}
	for _, root := range roots {
		if filepath.Dir(filepath.Clean(folder)) != filepath.Clean(root) {
			continue
		}
		resolved, err := fsx.EvalSymlinks(root)
		if err != nil {
			return InstallResult{}, err
		}
		if datadir.RealDirUnder(resolved, folder) {
			return s.store.InstallFolder(ctx, game, id, folder)
		}
	}
	return InstallResult{}, errors.New("that folder is not in an extra mods folder")
}

// InstallBuildFolder installs folder, one of the folders set to auto-install, into the profile as one local mod; a
// rebuild replaces the entry the previous install added, since both hold the same mods.
func (s *Service) InstallBuildFolder(ctx context.Context, game, id, folder string) (InstallResult, error) {
	if !slices.Contains(s.settings.Get().AutoInstalledFolders(), folder) {
		return InstallResult{}, errors.New("that folder is not set to auto-install")
	}
	return s.store.InstallFolder(ctx, game, id, folder)
}
