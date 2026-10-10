package profile

import (
	"context"
	"path/filepath"
)

// InstallItch unpacks an archive the player saved from itch.io game page `page` ("user/game") and adds it as a local
// file would be, with the page as its source, so the entry's page link and a share carry the page and no file.
func (s *Store) InstallItch(ctx context.Context, game, id, path, page string) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key, err := s.items.AddArchive(ctx, game, path)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	if err := s.namePackage(game, key, filepath.Base(path), ""); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, Source{Kind: KindItch, Name: page})
}
