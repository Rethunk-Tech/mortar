package profile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var zipFileUnsafe = strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")

// Service exposes the store to the frontend.
type Service struct {
	store    *Store
	home     string
	settings *settings.Store
	// App is set after application.New so export and restore can use native file dialogs.
	App *application.App
	// Version is Mortar's version written into exported zips.
	Version string
}

func NewService(store *Store, home string, settings *settings.Store) *Service {
	return &Service{store: store, home: home, settings: settings}
}

func (s *Service) gameModsDir(id string) (string, error) {
	dir, err := game.InstallDir(s.home, s.settings.Get(), id)
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

// History lists this profile's mod-set changes, newest first.
func (s *Service) History(game, id string) ([]HistoryEvent, error) {
	return s.store.History(game, id)
}

// Revert restores the profile's entries to the snapshot stored with eventID.
func (s *Service) Revert(game, id, eventID string) (Profile, error) {
	return s.store.Revert(game, id, eventID)
}

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

// SetAppearance replaces a profile's colour, icon and short description.
func (s *Service) SetAppearance(game, id, color, icon, description string) (Profile, error) {
	return s.store.SetAppearance(game, id, color, icon, description)
}

// SetLaunchOptions replaces a profile's extra SMAPI arguments.
func (s *Service) SetLaunchOptions(game, id, options string) (Profile, error) {
	return s.store.SetLaunchOptions(game, id, options)
}

// AddEntry copies the store item key into the profile.
func (s *Service) AddEntry(game, id, key string, source Source) (Profile, error) {
	return s.store.AddEntry(game, id, key, source)
}

// FomodPreview is the FOMOD wizard for a store item, given the choices so far.
func (s *Service) FomodPreview(game, id, key string, choices map[string]map[string][]string) (FomodAsk, error) {
	return s.store.FomodPreview(game, id, key, choices)
}

// FomodImage reads a file from the store item for the wizard; paths outside it are refused.
func (s *Service) FomodImage(game, key, rel string) ([]byte, error) {
	return s.store.FomodImage(game, key, rel)
}

// InstallFomod copies the store item into the profile using the chosen FOMOD plugins.
func (s *Service) InstallFomod(game, id, key string, source Source, choices map[string]map[string][]string) (InstallResult, error) {
	return s.store.InstallFomod(game, id, key, source, choices)
}

// InstallArchive unpacks the archive at path into the store and adds it to the profile.
func (s *Service) InstallArchive(game, id, path string) (InstallResult, error) {
	return s.store.InstallArchive(game, id, path)
}

func (s *Service) RemoveEntry(game, id, key string) (Profile, error) {
	return s.store.RemoveEntry(game, id, key)
}

func (s *Service) RemoveEntries(game, id string, keys []string) (Profile, error) {
	return s.store.RemoveEntries(game, id, keys)
}

// SetModEnabled switches a mod of the entry key on or off.
func (s *Service) SetModEnabled(game, id, key, uniqueID string, enabled bool) (EnableResult, error) {
	p, also, err := s.store.enableMod(game, id, key, uniqueID, enabled)
	return EnableResult{Profile: p, AlsoEnabled: also}, err
}

func (s *Service) SetModsEnabled(game, id string, mods []EnableRef, enabled bool) (EnableResult, error) {
	p, also, err := s.store.enableMods(game, id, mods, enabled)
	return EnableResult{Profile: p, AlsoEnabled: also}, err
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

// Mods lists the profile's mods. Unknown folders are parked so rebuild does not delete them; missing entry folders stay missing so drift can offer Restore.
func (s *Service) Mods(game, id string) ([]Mod, error) {
	missing, err := s.store.missingEntryFolders(game, id)
	if err != nil {
		return nil, err
	}
	if err := s.store.parkUnknownMods(game, id); err != nil {
		return nil, err
	}
	mods, err := s.store.UserMods(game, id)
	if err != nil {
		return nil, err
	}
	if err := s.store.unplaceKeys(game, id, missing); err != nil {
		return nil, err
	}
	return mods, nil
}

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

// ReadConfig returns the mod's config.json text.
func (s *Service) ReadConfig(game, id, key, uniqueID string) (string, error) {
	return s.store.ReadConfig(game, id, key, uniqueID)
}

// WriteConfig replaces the mod's config.json atomically.
func (s *Service) WriteConfig(game, id, key, uniqueID, contents string) error {
	return s.store.WriteConfig(game, id, key, uniqueID, contents)
}

// SetPinned records whether the entry stays on its current version.
func (s *Service) SetPinned(game, id, key string, pinned bool) (Profile, error) {
	return s.store.SetPinned(game, id, key, pinned)
}

// SetSkipVersion hides that exact newer version, or clears the skip when version is empty.
func (s *Service) SetSkipVersion(game, id, key, version string) (Profile, error) {
	return s.store.SetSkipVersion(game, id, key, version)
}

// ExportProfile asks where to save a zip of the whole profile and writes it. It returns "" when the dialog is cancelled.
func (s *Service) ExportProfile(game, id string) (string, error) {
	p, err := s.store.read(game, id)
	if err != nil {
		return "", err
	}
	if s.App == nil {
		return "", fmt.Errorf("no window")
	}
	d := s.App.Dialog.SaveFile()
	d.SetOptions(&application.SaveFileDialogOptions{
		Title:    "Export profile",
		Filename: zipFileUnsafe.Replace(p.Name) + ".zip",
	})
	d.AddFilter("Zip archive", "*.zip")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return dest, err
	}
	return dest, s.store.ExportZip(game, id, dest, s.Version)
}

// RestoreFromZip asks for a profile zip and imports it as a new profile. It returns an empty profile when cancelled.
func (s *Service) RestoreFromZip(game string) (Profile, error) {
	if s.App == nil {
		return Profile{}, fmt.Errorf("no window")
	}
	d := s.App.Dialog.OpenFile().
		SetTitle("Restore from zip").
		AddFilter("Zip archive", "*.zip")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return Profile{}, err
	}
	return s.store.RestoreZip(game, path)
}

// SetEntryNoteTags records the note and tags on one profile entry.
func (s *Service) SetEntryNoteTags(game, id, key, note string, tags []string) (Profile, error) {
	return s.store.SetEntryNoteTags(game, id, key, note, tags)
}
