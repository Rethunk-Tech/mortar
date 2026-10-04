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
	Version              string
	QueueProfileDeleted  func(game, id string)
	QueueProfileRestored func(game, id string)
}

func NewService(store *Store, home string, settings *settings.Store) *Service {
	store.NewModsEnabled = func() bool { return settings.Get().NewModsEnabled() }
	if settings != nil {
		store.OldFilesMode = func(game string) string { return settings.Get().GamePrefs(game).OldFilesOnUpdate }
	}
	store.home, store.settings = home, settings
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

func (s *Service) ListDamaged(game string) ([]Profile, error) { return s.store.ListDamaged(game) }

// Repair rebuilds a damaged profile.json from its latest history snapshot.
func (s *Service) Repair(game, id string) (Profile, error) { return s.store.Repair(game, id) }

// UndoRepair restores the aside damaged profile.json.
func (s *Service) UndoRepair(game, id string) error { return s.store.UndoRepair(game, id) }

// OpenFolder shows the profile's folder in the system file manager.
func (s *Service) OpenFolder(game, id string) error {
	dir, err := s.store.profileDir(game, id)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

// History lists this profile's mod-set changes, newest first.
func (s *Service) History(game, id string) ([]HistoryEvent, error) {
	return s.store.History(game, id)
}

// HealthHistory lists problem-check snapshots for this profile, oldest first.
func (s *Service) HealthHistory(game, id string) ([]HealthPoint, error) {
	return s.store.HealthHistory(game, id)
}

// RecentHistory lists the newest change events across this game's usable, non-hidden profiles.
func (s *Service) RecentHistory(game string) ([]RecentEvent, error) {
	return s.store.RecentHistory(game)
}

// Snapshot returns the entries for a history snapshot or event.
func (s *Service) Snapshot(game, id, snapshotID string) ([]Entry, error) {
	return s.store.Snapshot(game, id, snapshotID)
}

// Revert restores the profile's entries to the snapshot stored with eventID.
func (s *Service) Revert(game, id, eventID string) (Profile, error) {
	return s.store.Revert(game, id, eventID)
}

// ProfilesWithMod lists the profiles of game whose profile.json names uniqueID.
func (s *Service) ProfilesWithMod(game, uniqueID string) ([]ModInProfile, error) {
	return s.store.ProfilesWithMod(game, uniqueID)
}

// ModsByAuthor lists mods whose manifest Author field includes author in any profile of game.
func (s *Service) ModsByAuthor(game, author string) ([]AuthorMod, error) {
	return s.store.ModsByAuthor(game, author)
}

func (s *Service) Create(game, name string) (Profile, error) { return s.store.Create(game, name) }

func (s *Service) Rename(game, id, name string) (Profile, error) {
	return s.store.Rename(game, id, name)
}

func (s *Service) SetNotes(game, id, notes string) (Profile, error) {
	return s.store.SetNotes(game, id, notes)
}

// SetOverride sets one profile override of a game setting; "default" removes it so the game value applies.
func (s *Service) SetOverride(game, id, key, value string) (Profile, error) {
	return s.store.SetOverride(game, id, key, value)
}

func (s *Service) SetOverrides(game, id string, overrides map[string]string) (Profile, error) {
	return s.store.SetOverrides(game, id, overrides)
}

func (s *Service) SetSkipPlayCheck(game, id string, on bool) (Profile, error) {
	return s.store.SetSkipPlayCheck(game, id, on)
}

// SetAppearance replaces a profile's colour, icon and short description.
func (s *Service) SetAppearance(game, id, color, icon, description string) (Profile, error) {
	return s.store.SetAppearance(game, id, color, icon, description)
}

// SetLaunchOptions replaces a profile's extra SMAPI arguments.
func (s *Service) SetLaunchOptions(game, id, options string) (Profile, error) {
	return s.store.SetLaunchOptions(game, id, options)
}

// SetLaunchSettings replaces a profile's direct-launch prefix and environment.
func (s *Service) SetLaunchSettings(game, id, prefix, env string) (Profile, error) {
	return s.store.SetLaunchSettings(game, id, prefix, env)
}

// SetLaunchPresets replaces a profile's named launch presets and its default.
func (s *Service) SetLaunchPresets(game, id string, presets []LaunchPreset, defaultID string) (Profile, error) {
	return s.store.SetLaunchPresets(game, id, presets, defaultID)
}

// AddLaunchPreset appends a copy of preset to the profile's launch presets.
func (s *Service) AddLaunchPreset(game, id string, preset LaunchPreset) (Profile, error) {
	return s.store.AddLaunchPreset(game, id, preset)
}

// SetDefaultLaunchPreset marks the preset Play uses; empty selects the profile's own settings.
func (s *Service) SetDefaultLaunchPreset(game, id, presetID string) (Profile, error) {
	return s.store.SetDefaultLaunchPreset(game, id, presetID)
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

// SplitExtra turns an extra file of an entry into its own profile entry.
func (s *Service) SplitExtra(game, id, entryKey, extraKey string) (Profile, error) {
	return s.store.SplitExtra(game, id, entryKey, extraKey)
}

// CombineEntries attaches otherKey as an extra file of targetKey when both are from the same Nexus page.
func (s *Service) CombineEntries(game, id, targetKey, otherKey string) (Profile, error) {
	return s.store.CombineEntries(game, id, targetKey, otherKey)
}

func (s *Service) RemoveEntries(game, id string, keys []string) (Profile, error) {
	return s.store.RemoveEntries(game, id, keys)
}

// RestoreEntries puts removed store items back into the profile with their previous entry fields.
func (s *Service) RestoreEntries(game, id string, entries []Entry) (Profile, error) {
	return s.store.RestoreEntries(game, id, entries)
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
func (s *Service) Delete(game, id string) error {
	if err := s.store.Delete(game, id); err != nil {
		return err
	}
	if s.QueueProfileDeleted != nil {
		s.QueueProfileDeleted(game, id)
	}
	return nil
}

func (s *Service) ListTrash(game string) ([]TrashItem, error) { return s.store.ListTrash(game) }

func (s *Service) Restore(game, id string) (Profile, error) {
	p, err := s.store.Restore(game, id)
	if err == nil && s.QueueProfileRestored != nil {
		s.QueueProfileRestored(game, id)
	}
	return p, err
}

func (s *Service) Purge(game, id string) error { return s.store.Purge(game, id) }

func (s *Service) PurgeTrash(game string) error { return s.store.PurgeTrash(game) }

func (s *Service) SetHidden(game, id string, hidden bool) (Profile, error) {
	return s.store.SetHidden(game, id, hidden)
}

func (s *Service) Reorder(game string, ids []string) error { return s.store.Reorder(game, ids) }

// Mods lists the profile's mods. Missing entry folders stay missing so drift can offer Restore.
func (s *Service) Mods(game, id string) ([]Mod, error) {
	missing, err := s.store.missingEntryFolders(game, id)
	if err != nil {
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
func (s *Service) SetPinned(game, id, key string, pinned bool, pinReason string) (Profile, error) {
	return s.store.SetPinned(game, id, key, pinned, pinReason)
}

func (s *Service) SetPinnedMany(game, id string, keys []string, pinned bool, pinReason string) (Profile, error) {
	return s.store.SetPinnedMany(game, id, keys, pinned, pinReason)
}

// ClearCollection removes the Nexus collection link from a profile.
func (s *Service) ClearCollection(game, id string) (Profile, error) {
	return s.store.ClearCollection(game, id)
}

// SetCollection records the Nexus collection this profile was imported from.
func (s *Service) SetCollection(game, id string, ref CollectionRef) (Profile, error) {
	return s.store.SetCollection(game, id, ref)
}

// SetSkipVersion hides that exact newer version, or clears the skip when version is empty.
func (s *Service) SetSkipVersion(game, id, key, version string) (Profile, error) {
	return s.store.SetSkipVersion(game, id, key, version)
}

func (s *Service) SetSkipVersionMany(game, id string, refs []SkipVersionRef) (Profile, error) {
	return s.store.SetSkipVersionMany(game, id, refs)
}

// SetSkipSource records whether updates from source are hidden for the entry.
func (s *Service) SetSkipSource(game, id, key, source string, skip bool) (Profile, error) {
	return s.store.SetSkipSource(game, id, key, source, skip)
}

func (s *Service) SetSkipSourceMany(game, id string, keys []string, source string, skip bool) (Profile, error) {
	return s.store.SetSkipSourceMany(game, id, keys, source, skip)
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
	d.AttachToWindow(s.App.Window.Current())
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
	d.AttachToWindow(s.App.Window.Current())
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

// ListCustomCategories returns custom mod categories stored for game.
func (s *Service) ListCustomCategories(game string) ([]CustomCategory, error) {
	return s.store.ListCustomCategories(game)
}

// SaveCustomCategories replaces custom mod categories for game.
func (s *Service) SaveCustomCategories(game string, categories []CustomCategory) ([]CustomCategory, error) {
	return s.store.SaveCustomCategories(game, categories)
}

// SetEntryCategory sets or clears an entry's primary category override.
func (s *Service) SetEntryCategory(game, id, key, override string) (Profile, error) {
	return s.store.SetEntryCategory(game, id, key, override)
}

func (s *Service) SetEntryCategoryMany(game, id string, keys []string, override string) (Profile, error) {
	return s.store.SetEntryCategoryMany(game, id, keys, override)
}

func (s *Service) SetEntryTagsMany(game, id string, keys []string, tag string, add bool) (Profile, error) {
	return s.store.SetEntryTagsMany(game, id, keys, tag, add)
}

// RestoreEntryFields writes each entry's previous pin, skip, tags and category in one profile write.
func (s *Service) RestoreEntryFields(game, id string, fields []EntryFields) (Profile, error) {
	return s.store.RestoreEntryFields(game, id, fields)
}
