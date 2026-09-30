// Package loadersvc installs and checks each game's mod loader and keeps its bundled mods in every profile.
package loadersvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/loader"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ProgressEvent is emitted with a Progress after each install step.
const ProgressEvent = "loader:progress"

// Progress says which install step just finished for a game.
type Progress struct {
	Game string      `json:"game"`
	Step loader.Step `json:"step"`
}

// Service exposes loader status and install to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	items    *store.Store
	profiles *profile.Store
	busy     sync.Mutex
	// App is set after application.New so events can be emitted.
	App *application.App
}

func NewService(home string, s *settings.Store, items *store.Store, profiles *profile.Store) *Service {
	return &Service{home: home, settings: s, items: items, profiles: profiles}
}

// BundledKey returns the hook profile.Store uses to find the loader's bundled mods for the game, or "" when
// none are installed. It builds a missing store entry from the game folder first.
func BundledKey(s *Service) func(string) string {
	return func(id string) string {
		key, err := s.ensureBundled(id)
		if err != nil {
			log.Printf("bundled mods for %s: %v", id, err)
		}
		return key
	}
}

// SyncBundled makes sure the game's bundled mods are in the store and in every profile. It never fails the
// caller: the loader may be absent or the game not installed.
func SyncBundled(s *Service, id string) {
	key, err := s.ensureBundled(id)
	if err == nil && key != "" {
		err = s.profiles.ApplyBundled(id, key)
	}
	if err != nil {
		log.Printf("bundled mods for %s: %v", id, err)
	}
}

// ensureBundled returns the store key of the installed loader's bundled mods, building the entry from the game
// folder when the loader was installed outside Mortar. It returns "" when no loader is installed.
func (s *Service) ensureBundled(id string) (string, error) {
	st, err := s.LocalStatus(id)
	if err != nil {
		return "", err
	}
	if !st.Installed || st.Broken || st.Version == "" {
		return "", nil
	}
	key := store.SMAPIKey(st.Version)
	if _, err := s.items.Path(id, key); err == nil {
		return key, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	g, dir, err := s.target(id)
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "mortar-bundled-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := g.CopyBundled(dir, tmp); err != nil {
		return "", err
	}
	if err := s.items.AddDir(id, key, tmp); err != nil {
		return "", err
	}
	if s.settings.Get().Loaders[id] == "" {
		if _, err := s.settings.Update(func(v *settings.Settings) {
			v.Loaders = maps.Clone(v.Loaders)
			v.Loaders[id] = st.Version
		}); err != nil {
			return "", err
		}
	}
	return key, nil
}

func (s *Service) target(id string) (game.Game, string, error) {
	g := game.Find(id)
	if g == nil {
		return nil, "", fmt.Errorf("unknown game %q", id)
	}
	dir, err := game.InstallDir(s.home, s.settings.Get().GameFolders, id)
	if err != nil {
		return nil, "", err
	}
	if dir == "" {
		return nil, "", fmt.Errorf("%s is not installed", g.Name())
	}
	return g, dir, nil
}

// LocalStatus reports the loader's state on disk without any network call.
func (s *Service) LocalStatus(id string) (loader.Status, error) {
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	return g.LoaderStatus(dir, s.settings.Get().Loaders[id]), nil
}

// Status is LocalStatus plus whether a newer release exists. A failed release lookup only leaves Latest empty.
func (s *Service) Status(ctx context.Context, id string) (loader.Status, error) {
	g, _, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	st, err := s.LocalStatus(id)
	if err != nil {
		return loader.Status{}, err
	}
	if latest, err := g.LatestLoader(ctx); err == nil {
		st.Latest = latest
		st.UpdateAvailable = st.Installed && st.Version != "" && loader.Newer(latest, st.Version)
	}
	return st, nil
}

// Install installs the loader or, when it is already there, updates it, then puts its bundled mods in every profile.
func (s *Service) Install(ctx context.Context, id string) (loader.Status, error) {
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	if s.profiles.AnyRunning(id) {
		return loader.Status{}, fmt.Errorf("%s is running: close it before installing %s", g.Name(), g.LoaderName())
	}
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	bundled := func(version, modsDir string) error {
		key := store.SMAPIKey(version)
		if err := s.items.AddDir(id, key, modsDir); err != nil {
			return err
		}
		return s.profiles.ApplyBundled(id, key)
	}
	version, err := g.InstallLoader(ctx, dir, bundled, func(step loader.Step) {
		if s.App != nil {
			s.App.Event.Emit(ProgressEvent, Progress{Game: id, Step: step})
		}
	})
	if err != nil {
		return loader.Status{}, err
	}
	next, err := s.settings.Update(func(v *settings.Settings) {
		v.Loaders = maps.Clone(v.Loaders)
		v.Loaders[id] = version
	})
	if err != nil {
		return loader.Status{}, err
	}
	if s.App != nil {
		s.App.Event.Emit(settings.ChangedEvent, next)
	}
	return s.Status(ctx, id)
}
