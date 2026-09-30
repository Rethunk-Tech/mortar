package settings

import (
	"maps"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
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
	// ValidateImage vets a wallpaper path before it is stored; set before the app runs.
	ValidateImage func(path string) error
}

func NewService(store *Store) *Service { return &Service{store: store} }

func (s *Service) Get() Settings { return s.store.Get() }

func (s *Service) SetAccent(accent string) error {
	return s.set(func(v *Settings) { v.Accent = accent })
}

// SetBackupsKept sets how many save backups to retain.
func (s *Service) SetBackupsKept(n int) error {
	return s.set(func(v *Settings) { v.BackupsKept = n })
}

// SetCheckModUpdatesOnStart sets whether startup checks the last-opened profile of each game.
func (s *Service) SetCheckModUpdatesOnStart(on bool) error {
	return s.set(func(v *Settings) { v.CheckModUpdatesOnStart = &on })
}

// SetTellWhenSmapiOut sets whether Mortar toasts when a newer SMAPI exists.
func (s *Service) SetTellWhenSmapiOut(on bool) error {
	return s.set(func(v *Settings) { v.TellWhenSmapiOut = &on })
}

// SetListColumns stores which Mods list-view columns are shown.
func (s *Service) SetListColumns(ids []string) error {
	return s.set(func(v *Settings) { v.ListColumns = ids })
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

func (s *Service) SetBackground(background string) error {
	return s.set(func(v *Settings) { v.Background = background })
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
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
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

// SetGameFolder stores dir as the game's install folder, or clears the override when dir is empty.
func (s *Service) SetGameFolder(game, dir string) error {
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

// ChooseGameFolder asks for a folder and stores it as the game's install folder; cancelling changes nothing.
func (s *Service) ChooseGameFolder(game string) error {
	d := s.App.Dialog.OpenFile().
		SetTitle("Choose game folder").
		CanChooseDirectories(true).
		CanChooseFiles(false)
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	dir, err := d.PromptForSingleSelection()
	if err != nil || dir == "" {
		return err
	}
	return s.SetGameFolder(game, dir)
}

// DataFolder is Mortar's data folder and the space it takes.
type DataFolder struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// DataFolder measures the data folder; Wails runs it off the UI thread.
func (s *Service) DataFolder() (DataFolder, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return DataFolder{}, err
	}
	size, err := datadir.Size(dir)
	return DataFolder{Path: dir, Size: size}, err
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
	d := s.App.Dialog.SaveFile()
	d.SetOptions(&application.SaveFileDialogOptions{Title: "Export settings", Filename: "mortar-settings.json"})
	d.AddFilter("JSON", "*.json")
	d.AddFilter("All files", "*")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, body, 0o600)
}

// PreviewImportSettings opens a JSON file, validates it, and returns what would change. Empty Raw means cancelled.
func (s *Service) PreviewImportSettings() (ImportPreview, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle("Import settings").
		AddFilter("JSON", "*.json").
		AddFilter("All files", "*")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return ImportPreview{}, err
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return ImportPreview{}, err
	}
	p, present, err := ParseExport(b)
	if err != nil {
		return ImportPreview{}, err
	}
	return ImportPreview{Raw: string(b), Changes: previewChanges(s.store.Get(), p, present)}, nil
}

// ApplyImportedSettings applies a previously previewed export. Unknown fields stay ignored.
func (s *Service) ApplyImportedSettings(raw string) error {
	p, present, err := ParseExport([]byte(raw))
	if err != nil {
		return err
	}
	return s.set(func(v *Settings) {
		ApplyExport(v, p, present)
	})
}

func (s *Service) set(fn func(*Settings)) error {
	next, err := s.store.Update(fn)
	if err != nil {
		return err
	}
	if s.App != nil {
		s.App.Event.Emit(ChangedEvent, next)
	}
	return nil
}
