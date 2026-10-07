package settings

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/picker"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ChangedEvent is emitted with the new Settings after every successful setter.
const ChangedEvent = "settings:changed"

// Service exposes the store to the frontend.
type Service struct {
	store *Store
	// App is set after application.New so setters can emit events.
	App *application.App
	// ValidateGameFolder vets a folder before it is stored; set before the app runs.
	ValidateGameFolder func(game, dir string) error
	// ValidateLauncherRoot vets a launcher folder before it is stored; set before the app runs.
	ValidateLauncherRoot func(launcher, dir string) error
	// ValidateImage vets a wallpaper path before it is stored; set before the app runs.
	ValidateImage func(path string) error
}

func NewService(store *Store) *Service { return &Service{store: store} }

func (s *Service) Get() Settings { return s.store.Get() }

func (s *Service) SetLanguage(language string) error {
	return s.set(func(v *Settings) { v.Language = language })
}

func (s *Service) SetAccent(accent string) error {
	return s.set(func(v *Settings) { v.Accent = accent })
}

func (s *Service) SetTellWhenSmapiOut(on bool) error {
	return s.set(func(v *Settings) { v.TellWhenSmapiOut = &on })
}

func (s *Service) SetKeepInTray(on bool) error {
	return s.set(func(v *Settings) { v.KeepInTray = on })
}

func (s *Service) SetLanSharing(on bool) error {
	return s.set(func(v *Settings) { v.LanSharing = on })
}

func (s *Service) SetLanPort(port int) error {
	return s.set(func(v *Settings) { v.LanPort = port })
}

func (s *Service) SetIncludeBetaReleases(on bool) error {
	return s.set(func(v *Settings) { v.IncludeBetaReleases = on })
}

func (s *Service) SetIncludePrereleaseModVersions(on bool) error {
	return s.set(func(v *Settings) { v.IncludePrereleaseModVersions = on })
}

func (s *Service) SetCheckOnlyEnabledMods(on bool) error {
	return s.set(func(v *Settings) { v.CheckOnlyEnabledMods = on })
}

func (s *Service) SetEnableModsWhenInstalled(on bool) error {
	return s.set(func(v *Settings) { v.EnableModsWhenInstalled = &on })
}

func (s *Service) SetBackground(background string) error {
	return s.set(func(v *Settings) { v.Background = background })
}

func (s *Service) SetNexusPreferredDownloadServer(shortName string) error {
	return s.set(func(v *Settings) { v.NexusPreferredDownloadServer = shortName })
}

func (s *Service) SetNxmRedirectOtherGames(on bool) error {
	return s.set(func(v *Settings) { v.NxmRedirectOtherGames = &on })
}

// CorruptSettingsPath returns the one-time path of settings preserved at startup.
func (s *Service) CorruptSettingsPath() string { return s.store.CorruptPath() }

// ShowCorruptSettings opens the folder holding the damaged settings file kept at startup.
func (s *Service) ShowCorruptSettings() error {
	path := s.store.CorruptPath()
	if path == "" {
		return nil
	}
	return datadir.Open(filepath.Dir(path))
}

// SetListColumns stores which Mods list-view columns are shown for the game.
func (s *Service) SetListColumns(game string, ids []string) error {
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		gp.ListColumns = ids
		putGame(v, game, gp)
	})
}

// SetListSort stores the Mods list-view sort column and direction.
func (s *Service) SetListSort(column, dir string) error {
	return s.set(func(v *Settings) { v.ListSortColumn, v.ListSortDir = column, dir })
}

// SetListGroupBy stores how the Mods tab groups the list and grid.
func (s *Service) SetListGroupBy(by string) error {
	return s.set(func(v *Settings) { v.ListGroupBy = by })
}

// SetTipsSeen stores which empty-state tips the user has dismissed.
func (s *Service) SetTipsSeen(ids []string) error {
	return s.set(func(v *Settings) { v.TipsSeen = ids })
}

// SetSmapiToastAt stores when Mortar last showed the SMAPI-update toast.
func (s *Service) SetSmapiToastAt(at string) error {
	return s.set(func(v *Settings) { v.SmapiToastAt = at })
}

// SetBackgroundImage stores path as the wallpaper, or restores the default one when path is empty.
func (s *Service) SetBackgroundImage(path string) error {
	if path != "" && s.ValidateImage != nil {
		if err := s.ValidateImage(path); err != nil {
			return err
		}
	}
	return s.set(func(v *Settings) { v.BackgroundImage = path })
}

// ChooseBackgroundImage asks for an image and stores it as the wallpaper; cancelling changes nothing.
func (s *Service) ChooseBackgroundImage() error {
	d := s.App.Dialog.OpenFile().
		SetTitle("Choose background image").
		AddFilter("Images (PNG, JPEG, WebP)", "*.png;*.jpg;*.jpeg;*.webp").
		AddFilter("All files", "*")
	d.AttachToWindow(s.App.Window.Current())
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return err
	}
	return s.SetBackgroundImage(path)
}

func (s *Service) SetLastGame(game string) error {
	return s.set(func(v *Settings) { v.LastGame = game })
}

func (s *Service) SetLastProfile(game, id string) error {
	return s.set(func(v *Settings) {
		v.LastProfile = maps.Clone(v.LastProfile)
		if v.LastProfile == nil {
			v.LastProfile = map[string]string{}
		}
		v.LastProfile[game] = id
	})
}

// SetGameFolder stores dir as the game's install folder, or clears the override when dir is empty. The folder is kept
// cleaned: a trailing separator would reach installers' command lines as part of the path.
func (s *Service) SetGameFolder(game, dir string) error {
	if dir != "" {
		dir = filepath.Clean(dir)
	}
	if s.ValidateGameFolder != nil {
		if err := s.ValidateGameFolder(game, dir); err != nil {
			return err
		}
	}
	return s.set(func(v *Settings) {
		v.GameFolders = maps.Clone(v.GameFolders)
		if v.GameFolders == nil {
			v.GameFolders = map[string]string{}
		}
		if dir == "" {
			delete(v.GameFolders, game)
		} else {
			v.GameFolders[game] = dir
		}
	})
}

// AddLauncherRoot adds a folder of a launcher for Mortar to search, after checking it is that launcher's.
func (s *Service) AddLauncherRoot(launcher, dir string) error {
	if s.ValidateLauncherRoot != nil {
		if err := s.ValidateLauncherRoot(launcher, dir); err != nil {
			return err
		}
	}
	return s.set(func(v *Settings) {
		v.LauncherRoots = maps.Clone(v.LauncherRoots)
		if v.LauncherRoots == nil {
			v.LauncherRoots = map[string][]string{}
		}
		if !slices.Contains(v.LauncherRoots[launcher], dir) {
			v.LauncherRoots[launcher] = append(slices.Clone(v.LauncherRoots[launcher]), dir)
		}
	})
}

// RemoveLauncherRoot stops searching a folder the user added for a launcher.
func (s *Service) RemoveLauncherRoot(launcher, dir string) error {
	return s.set(func(v *Settings) {
		v.LauncherRoots = maps.Clone(v.LauncherRoots)
		rest := slices.DeleteFunc(slices.Clone(v.LauncherRoots[launcher]), func(d string) bool { return d == dir })
		if len(rest) == 0 {
			delete(v.LauncherRoots, launcher)
		} else {
			v.LauncherRoots[launcher] = rest
		}
	})
}

// ConfirmLaunchers records that first run's launcher screen is done.
func (s *Service) ConfirmLaunchers() error {
	return s.set(func(v *Settings) { v.LaunchersConfirmed = true })
}

// SetGameStore stores the chosen store for game, or clears it when store is empty.
func (s *Service) SetGameStore(game, store string) error {
	switch store {
	case "", "steam", "flatpak-steam", "gog", "gog-heroic", "gog-minigalaxy", "lutris", "bottles":
	default:
		return fmt.Errorf("unknown store %q", store)
	}
	return s.set(func(v *Settings) {
		v.GameStores = maps.Clone(v.GameStores)
		if v.GameStores == nil {
			v.GameStores = map[string]string{}
		}
		if store == "" {
			delete(v.GameStores, game)
			return
		}
		v.GameStores[game] = store
	})
}

// ChooseGameFolder asks for a folder and stores it as the game's install folder; cancelling changes nothing.
func (s *Service) ChooseGameFolder(game string) error {
	d := s.App.Dialog.OpenFile().
		SetTitle("Choose game folder").
		CanChooseDirectories(true).
		CanChooseFiles(false)
	d.AttachToWindow(s.App.Window.Current())
	dir, err := d.PromptForSingleSelection()
	if err != nil || dir == "" {
		return err
	}
	return s.SetGameFolder(game, dir)
}

// OpenDataFolder shows the data folder in the system file manager.
func (s *Service) OpenDataFolder() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

// ExportSettings writes a redacted settings JSON through the native save dialog. It returns "" when cancelled.
func (s *Service) ExportSettings() (string, error) {
	body, err := MarshalExport(s.store.Get())
	if err != nil {
		return "", err
	}
	return picker.SaveFile(s.App, "Export settings", "mortar-settings.json", "JSON", "*.json", body)
}

// PickImportFile asks for a settings file to import. It returns "" when cancelled.
func (s *Service) PickImportFile() (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle("Import settings").
		AddFilter("JSON", "*.json").
		AddFilter("All files", "*")
	d.AttachToWindow(s.App.Window.Current())
	return d.PromptForSingleSelection()
}

// PreviewImport validates the export at path and lists, per section, what importing it would change.
func (s *Service) PreviewImport(path string) (ImportPreview, error) {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return ImportPreview{}, err
	}
	return PreviewImport(s.store.Get(), b)
}

// ApplyImport applies the export at path, only the fields in the chosen sections.
func (s *Service) ApplyImport(path string, sections []string) error {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	var applyErr error
	err = s.set(func(v *Settings) { applyErr = ApplyImport(v, b, sections) })
	if applyErr != nil {
		return applyErr
	}
	return err
}

// SetAskEndorseMods sets whether Mortar suggests endorsing mods after clean runs.
func (s *Service) SetAskEndorseMods(on bool) error {
	return s.set(func(v *Settings) { v.AskEndorseMods = &on })
}

// PrefSpecs returns the registry descriptor for the frontend and CLI.
func (s *Service) PrefSpecs() []PrefSpec { return PrefSpecs() }

// SetByKey writes one CLI-visible setting.
func (s *Service) SetByKey(key, value, game string) error {
	var applyErr error
	err := s.setReapplying(key == "launchAtLogin", func(cur *Settings) {
		applyErr = ApplyKeyGame(cur, key, value, game)
	})
	if applyErr != nil {
		return applyErr
	}
	return err
}

// SetShortcuts stores keyboard chords by action id. Missing ids keep their defaults.
func (s *Service) SetShortcuts(chords map[string]string) error {
	return s.set(func(v *Settings) {
		v.Shortcuts = maps.Clone(chords)
		if v.Shortcuts == nil {
			v.Shortcuts = map[string]string{}
		}
	})
}

func (s *Service) set(fn func(*Settings)) error { return s.setReapplying(false, fn) }

// setReapplying persists fn's change, then brings the sign-in autostart entry in line with launchAtLogin when that
// changed or force is set, so an accepted write has its effect and a lost entry is rewritten by setting true again.
func (s *Service) setReapplying(force bool, fn func(*Settings)) error {
	var before bool
	next, err := s.store.Update(func(v *Settings) {
		before = v.LaunchAtLogin
		fn(v)
	})
	if err != nil {
		return err
	}
	var autostartErr error
	if force || next.LaunchAtLogin != before {
		if err := applyAutostart(next.LaunchAtLogin); err != nil {
			autostartErr = fmt.Errorf("saved, but the sign-in autostart entry was not updated: %w", err)
		}
	}
	if s.App != nil {
		s.App.Event.Emit(ChangedEvent, next)
	}
	return autostartErr
}
