package profile

import (
	"errors"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

func (s *Service) extraModsFolder(game string) string {
	return s.settings.Get().GamePrefs(game).ExtraModsFolder
}

// ExtraFolderMods lists the mods in the game's extra mods folder (setting extraModsFolder) as Import from the game's
// Mods folder previews them; an unset or missing folder lists none.
func (s *Service) ExtraFolderMods(game string) (GameModsPreview, error) {
	dir := s.extraModsFolder(game)
	if dir == "" {
		return GameModsPreview{Mods: []GameModPreview{}}, nil
	}
	return s.store.PreviewGameMods(dir)
}

// InstallExtraFolderMod installs folder, a top-level folder of the extra mods folder as ExtraFolderMods returned it,
// into the profile the way a dropped folder is installed.
func (s *Service) InstallExtraFolderMod(game, id, folder string) (InstallResult, error) {
	root := s.extraModsFolder(game)
	if root == "" {
		return InstallResult{}, errors.New("no extra mods folder is set")
	}
	resolved, err := fsx.EvalSymlinks(root)
	if err != nil {
		return InstallResult{}, err
	}
	if filepath.Dir(filepath.Clean(folder)) != filepath.Clean(root) || !datadir.RealDirUnder(resolved, folder) {
		return InstallResult{}, errors.New("that folder is not in the extra mods folder")
	}
	return s.store.InstallFolder(game, id, folder)
}
