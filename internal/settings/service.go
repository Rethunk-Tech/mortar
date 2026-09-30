package settings

import (
	"maps"

	"github.com/Rethunk-AI/mortar/internal/datadir"
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
