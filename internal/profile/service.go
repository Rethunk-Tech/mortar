package profile

import (
	"fmt"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Service exposes the store to the frontend.
type Service struct {
	store    *Store
	home     string
	settings *settings.Store
}

func NewService(store *Store, home string, settings *settings.Store) *Service {
	return &Service{store: store, home: home, settings: settings}
}

func (s *Service) gameModsDir(id string) (string, error) {
	dir, err := game.InstallDir(s.home, s.settings.Get().GameFolders, id)
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("%s is not installed", id)
	}
	return filepath.Join(dir, "Mods"), nil
}

// PreviewGameMods lists mods in the game folder's Mods that ImportGameMods would copy.
func (s *Service) PreviewGameMods(gameID string) (GameModsPreview, error) {
	dir, err := s.gameModsDir(gameID)
	if err != nil {
		return GameModsPreview{}, err
	}
	return s.store.PreviewGameMods(dir)
}

// ImportGameMods copies the game folder's Mods into a new profile without changing that folder.
func (s *Service) ImportGameMods(gameID string) (GameModsResult, error) {
	dir, err := s.gameModsDir(gameID)
	if err != nil {
		return GameModsResult{}, err
	}
	return s.store.ImportGameMods(gameID, dir)
}

func (s *Service) List(game string) ([]Profile, error) { return s.store.List(game) }

// ProfilesWithMod lists the profiles of game whose profile.json names uniqueID.
func (s *Service) ProfilesWithMod(game, uniqueID string) ([]ModInProfile, error) {
	return s.store.ProfilesWithMod(game, uniqueID)
}

func (s *Service) Create(game, name string) (Profile, error) { return s.store.Create(game, name) }

func (s *Service) Rename(game, id, name string) (Profile, error) {
	return s.store.Rename(game, id, name)
}

func (s *Service) SetNotes(game, id, notes string) (Profile, error) {
	return s.store.SetNotes(game, id, notes)
}

// AddEntry copies the store item key into the profile.
func (s *Service) AddEntry(game, id, key string, source Source) (Profile, error) {
	return s.store.AddEntry(game, id, key, source)
}

// InstallArchive unpacks the archive at path into the store and adds it to the profile.
func (s *Service) InstallArchive(game, id, path string) (InstallResult, error) {
	return s.store.InstallArchive(game, id, path)
}

func (s *Service) RemoveEntry(game, id, key string) (Profile, error) {
	return s.store.RemoveEntry(game, id, key)
}

// SetModEnabled switches a mod of the entry key on or off.
func (s *Service) SetModEnabled(game, id, key, uniqueID string, enabled bool) (Profile, error) {
	return s.store.SetModEnabled(game, id, key, uniqueID, enabled)
}

// SetCover copies the image at path into the profile as its hero cover.
func (s *Service) SetCover(game, id, path string) (Profile, error) {
	return s.store.SetCover(game, id, path)
}

// ClearCover goes back to the automatic cover.
func (s *Service) ClearCover(game, id string) (Profile, error) { return s.store.ClearCover(game, id) }

// Covers lists the hero images to try, in order.
func (s *Service) Covers(game, id string) ([]string, error) { return s.store.Covers(game, id) }

func (s *Service) Duplicate(game, id string) (Profile, error) { return s.store.Duplicate(game, id) }

// Diff compares two profiles of the same game by UniqueID.
func (s *Service) Diff(game, aID, bID string) (Diff, error) { return s.store.Diff(game, aID, bID) }

// CopyMods copies selected mods from one profile into another from the store, with no download.
func (s *Service) CopyMods(game, fromID, toID string, uniqueIDs []string) (Profile, error) {
	return s.store.CopyMods(game, fromID, toID, uniqueIDs)
}

// Delete moves the profile to the trash, where it stays restorable for 30 days.
func (s *Service) Delete(game, id string) error { return s.store.Delete(game, id) }

func (s *Service) ListTrash(game string) ([]TrashItem, error) { return s.store.ListTrash(game) }

func (s *Service) Restore(game, id string) (Profile, error) { return s.store.Restore(game, id) }

func (s *Service) SetHidden(game, id string, hidden bool) (Profile, error) {
	return s.store.SetHidden(game, id, hidden)
}

func (s *Service) Reorder(game string, ids []string) error { return s.store.Reorder(game, ids) }

// Mods lists the profile's mods, rebuilding missing mod folders from the store first.
func (s *Service) Mods(game, id string) ([]Mod, error) { return s.store.UserMods(game, id) }

// ShowFiles opens the folder of the mod in entry key in the system file manager.
func (s *Service) ShowFiles(game, id, key, uniqueID string) error {
	dir, err := s.store.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

// OpenConfig opens the mod's config.json with the default app.
func (s *Service) OpenConfig(game, id, key, uniqueID string) error {
	path, err := s.store.ConfigPath(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	return datadir.Open(path)
}

// ModState reads the mod's rollback target and the state of its config.json.
func (s *Service) ModState(game, id, key, uniqueID string) (ModState, error) {
	return s.store.ModState(game, id, key, uniqueID)
}

// ResetConfig deletes the mod's config.json so the mod regenerates it.
func (s *Service) ResetConfig(game, id, key, uniqueID string) error {
	return s.store.ResetConfig(game, id, key, uniqueID)
}
