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
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/loader"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ProgressEvent is emitted with a Progress after each install step.
const ProgressEvent = "loader:progress"

// StateEvent is emitted with a State when a loader install starts and when it ends.
const StateEvent = "loader:state"

// State says whether a game's loader install is running. Error is set on the event that ends a failed one.
type State struct {
	Game       string `json:"game"`
	Installing bool   `json:"installing"`
	Error      string `json:"error"`
}

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
	// background tracks installs started by ensureInBackground.
	background sync.WaitGroup
	// run installs or updates the loader with busy held; tests replace it.
	run func(ctx context.Context, id string) (loader.Status, error)
	// App is set after application.New so events can be emitted.
	App *application.App
	// procDir is where running processes are listed on Linux; tests point it at a fake.
	procDir string
}

func NewService(home string, s *settings.Store, items *store.Store, profiles *profile.Store) *Service {
	svc := &Service{home: home, settings: s, items: items, profiles: profiles, procDir: "/proc"}
	svc.run = svc.install
	return svc
}

// Attach wires the service into the profile store: every profile gets the game's bundled mods, and creating a
// profile installs a missing loader in the background. It also syncs the bundled mods into the existing profiles.
func Attach(s *Service, id string) {
	s.profiles.Bundled = s.bundles
	s.profiles.Created = s.ensureInBackground
	SyncBundled(s, id)
}

// EnsureExisting installs the game's loader in the background when profiles exist and it is missing or broken.
// Call it once s.App is set, so the install's progress reaches the window.
func EnsureExisting(s *Service, id string) {
	if all, err := s.profiles.List(id); err == nil && len(all) > 0 {
		s.ensureInBackground(id)
	}
}

// bundles returns the store items every profile of the game holds, building any that is missing.
func (s *Service) bundles(id string) []profile.Bundle {
	var out []profile.Bundle
	if key, err := s.ensureBundled(id); err != nil {
		log.Printf("bundled mods for %s: %v", id, err)
	} else if key != "" {
		out = append(out, profile.Bundle{Key: key, Source: profile.Source{Kind: profile.SourceSMAPI, Name: "SMAPI"}})
	}
	if b, err := s.ensureBridge(id); err != nil {
		log.Printf("bridge for %s: %v", id, err)
	} else if b.Key != "" {
		out = append(out, b)
	}
	return out
}

// SyncBundled makes sure the game's bundled mods are in the store and in every profile. It never fails the
// caller: the loader may be absent or the game not installed.
func SyncBundled(s *Service, id string) {
	for _, b := range s.bundles(id) {
		if err := s.profiles.ApplyBundled(id, b); err != nil {
			log.Printf("bundled mods for %s: %v", id, err)
		}
	}
}

// ensureBridge returns the bundled console bridge's entry, adding it to the store first when it is missing.
// The bridge needs neither the loader nor the game folder, so every profile has it from creation.
func (s *Service) ensureBridge(id string) (profile.Bundle, error) {
	g := game.Find(id)
	if g == nil || g.BridgeVersion() == "" {
		return profile.Bundle{}, nil
	}
	key := store.BridgeKey(g.BridgeVersion())
	b := profile.Bundle{Key: key, Source: profile.Source{Kind: profile.SourceMortar, Name: "Mortar"}}
	if _, err := s.items.Path(id, key); err == nil {
		return b, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return profile.Bundle{}, err
	}
	tmp, err := os.MkdirTemp("", "mortar-bridge-")
	if err != nil {
		return profile.Bundle{}, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := g.ExtractBridge(tmp); err != nil {
		return profile.Bundle{}, err
	}
	return b, s.items.AddDir(id, key, tmp)
}

// ensureInBackground installs the game's loader when it is missing or broken, without blocking the caller.
// Failures are logged and announced with StateEvent; the game may simply not be installed.
func (s *Service) ensureInBackground(id string) {
	s.background.Go(func() {
		if _, err := s.Ensure(context.Background(), id); err != nil {
			log.Printf("loader for %s: %v", id, err)
		}
	})
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
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	return s.run(ctx, id)
}

// Ensure installs the loader when it is missing or broken, waiting for an install already running, and does
// nothing when it is fine. A newer release is never applied here: an update can break mods, so the user does it.
func (s *Service) Ensure(ctx context.Context, id string) (loader.Status, error) {
	s.busy.Lock()
	defer s.busy.Unlock()
	st, err := s.LocalStatus(id)
	if err != nil {
		return loader.Status{}, err
	}
	if st.Installed && !st.Broken {
		return st, nil
	}
	return s.run(ctx, id)
}

// install runs one install with busy held and announces its start and end.
func (s *Service) install(ctx context.Context, id string) (st loader.Status, err error) {
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	running, err := s.gameRunning(g)
	if err != nil {
		return loader.Status{}, err
	}
	if running || s.profiles.AnyRunning(id) {
		return loader.Status{}, fmt.Errorf("%s is running: close it before installing %s", g.Name(), g.LoaderName())
	}
	s.emit(StateEvent, State{Game: id, Installing: true})
	defer func() {
		state := State{Game: id}
		if err != nil {
			state.Error = err.Error()
		}
		s.emit(StateEvent, state)
	}()
	bundled := func(version, modsDir string) error {
		key := store.SMAPIKey(version)
		if err := s.items.AddDir(id, key, modsDir); err != nil {
			return err
		}
		return s.profiles.ApplyBundled(id, profile.Bundle{Key: key, Source: profile.Source{Kind: profile.SourceSMAPI, Name: "SMAPI"}})
	}
	version, err := g.InstallLoader(ctx, dir, bundled, func(step loader.Step) {
		s.emit(ProgressEvent, Progress{Game: id, Step: step})
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
	s.emit(settings.ChangedEvent, next)
	return s.Status(ctx, id)
}

// gameRunning reports whether any process of the game runs, with or without its loader and however it was started.
func (s *Service) gameRunning(g game.Game) (bool, error) {
	for _, name := range g.GameProcesses() {
		procs, err := launch.Processes(s.procDir, name)
		if err != nil {
			return false, fmt.Errorf("check for a running %s: %w", g.Name(), err)
		}
		if len(procs) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
	}
}
