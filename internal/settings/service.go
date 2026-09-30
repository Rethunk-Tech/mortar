package settings

import (
	"maps"

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
}

func NewService(store *Store) *Service { return &Service{store: store} }

func (s *Service) Get() Settings { return s.store.Get() }

func (s *Service) SetAccent(accent string) error {
	return s.set(func(v *Settings) { v.Accent = accent })
}

func (s *Service) SetTranslucent(on bool) error {
	return s.set(func(v *Settings) { v.Translucent = on })
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
