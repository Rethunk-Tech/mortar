// Package loadersvc installs and checks each game's mod loader and keeps its bundled mods in every profile.
package loadersvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	home       string
	settings   *settings.Store
	items      *store.Store
	profiles   *profile.Store
	components *components.Client
	busy       sync.Mutex
	// background tracks installs started by ensureInBackground.
	background sync.WaitGroup
	// run installs or updates the loader with busy held; tests replace it.
	run func(ctx context.Context, id string, fromStart bool) (loader.Status, error)
	// App is set after application.New so events can be emitted.
	App *application.App
	// OnReady runs after a successful loader install (SMAPI version may have changed).
	OnReady func(id string)
	// procDir is where running processes are listed on Linux; tests point it at a fake.
	procDir string
	// listVersions and fetchInstall let tests supply a release list and skip the installer.
	listVersions func(context.Context, string) ([]string, error)
	fetchInstall func(context.Context, string, string) error
}

func NewService(home string, s *settings.Store, items *store.Store, profiles *profile.Store, clients ...*components.Client) *Service {
	var client *components.Client
	if len(clients) > 0 {
		client = clients[0]
	}
	svc := &Service{home: home, settings: s, items: items, profiles: profiles, components: client, procDir: "/proc"}
	game.ConfigureComponents(client)
	svc.run = svc.install
	return svc
}

// Attach wires the service into the profile store: every profile gets the game's bundled mods, and creating a
// profile installs a missing loader in the background. It also syncs the bundled mods into the existing profiles.
func Attach(s *Service) {
	s.profiles.Bundled = s.bundles
	s.profiles.Created = s.ensureInBackground
}

// EnsureExisting installs the game's loader in the background when profiles exist and it is missing or broken.
// Call it once s.App is set, so the install's progress reaches the window.
func EnsureExisting(s *Service, id string) {
	if all, err := s.profiles.List(id); err == nil {
		for _, p := range all {
			if p.Error == "" {
				s.ensureInBackground(id)
				break
			}
		}
	}
}

// bundles returns the store items every profile of the game holds, building any that is missing.
func (s *Service) bundles(id string) []profile.Bundle {
	var out []profile.Bundle
	if key, err := s.ensureBundled(id); err != nil {
		log.Printf("bundled mods for %s: %v", id, err)
	} else if key != "" {
		if b, ok := s.bundleOf(id, key); ok {
			out = append(out, b)
		}
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

// ensureBridge returns the manifest's console bridge entry, adding it to the store first when it is missing.
// The bridge needs neither the loader nor the game folder, so every profile has it from creation.
func (s *Service) ensureBridge(id string) (profile.Bundle, error) {
	if s.components == nil {
		return profile.Bundle{}, nil
	}
	component, ok := s.components.Component(id, "bridge")
	if !ok || component.Kind != "bridge" {
		return profile.Bundle{}, nil
	}
	key := store.BridgeKey(component.Version, component.SHA256)
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
	defer func() { _ = fsx.RemoveAll(tmp) }()
	archivePath := filepath.Join(tmp, "bridge.zip")
	if err := s.components.Download(context.Background(), component, archivePath); err != nil {
		return profile.Bundle{}, err
	}
	unpacked := filepath.Join(tmp, "unpacked")
	if err := os.Mkdir(unpacked, 0o700); err != nil {
		return profile.Bundle{}, err
	}
	if err := archive.Extract(archivePath, unpacked); err != nil {
		return profile.Bundle{}, err
	}
	return b, s.items.AddDir(id, key, unpacked)
}

// ensureInBackground installs the game's loader when it is missing or broken, without blocking the caller.
// Failures are logged and announced with StateEvent; the game may simply not be installed.
func (s *Service) ensureInBackground(id string) {
	s.background.Go(func() {
		if _, err := s.Ensure(context.Background(), id, false); err != nil {
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
	l, ok := game.PrimaryLoader(id)
	if !ok {
		return "", nil
	}
	key := store.LoaderKey(l.ID(), st.Version)
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
	defer func() { _ = fsx.RemoveAll(tmp) }()
	copier, canCopy := l.(loader.BundledCopier)
	if !canCopy {
		return "", fmt.Errorf("%s ships no mods of its own", g.Name())
	}
	if err := copier.CopyBundled(dir, tmp); err != nil {
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

// catalogGame is the game's catalog entry; the zero value when the catalog lacks it.
func catalogGame(id string) components.GameInfo {
	for _, g := range game.Catalog() {
		if g.ID == id {
			return g
		}
	}
	return components.GameInfo{}
}

func (s *Service) target(id string) (game.Game, string, error) {
	g, err := game.Require(id)
	if err != nil {
		return nil, "", err
	}
	dir, err := game.InstallDir(s.home, s.settings.Get(), id)
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
	_, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	return game.LoaderStatus(id, dir, s.settings.Get().Loaders[id])
}

// Status is LocalStatus plus whether a newer release exists. A failed release lookup only leaves Latest empty.
func (s *Service) Status(ctx context.Context, id string) (loader.Status, error) {
	if _, _, err := s.target(id); err != nil {
		return loader.Status{}, err
	}
	st, err := s.LocalStatus(id)
	if err != nil {
		return loader.Status{}, err
	}
	if pin := s.settings.Get().SmapiPin; pin != "" {
		return st, nil
	}
	l, _ := game.PrimaryLoader(id)
	rel, ok := l.(loader.Releases)
	if !ok {
		return st, nil
	}
	if latest, err := rel.Latest(ctx, catalogGame(id)); err == nil {
		st.Latest = latest
		st.UpdateAvailable = st.Installed && st.Version != "" && meta.Newer(latest, st.Version)
	}
	return st, nil
}

// Install installs the loader or, when it is already there, updates it, then puts its bundled mods in every profile.
func (s *Service) Install(ctx context.Context, id string) (loader.Status, error) {
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	return s.run(ctx, id, false)
}

// Ensure installs the loader when it is missing or broken, waiting for an install already running, and does
// nothing when it is fine. A newer release is never applied here: an update can break mods, so the user does it.
// fromStart is true when Play requested this install, so a preparing claim for that Start is not treated as running.
//
//wails:ignore
func (s *Service) Ensure(ctx context.Context, id string, fromStart bool) (loader.Status, error) {
	s.busy.Lock()
	defer s.busy.Unlock()
	st, err := s.LocalStatus(id)
	if err != nil {
		return loader.Status{}, err
	}
	pin := s.settings.Get().SmapiPin
	if st.Installed && !st.Broken && (pin == "" || st.Version == pin) {
		return st, nil
	}
	return s.run(ctx, id, fromStart)
}

// install runs one install with busy held and announces its start and end.
func (s *Service) install(ctx context.Context, id string, fromStart bool) (st loader.Status, err error) {
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	running, err := s.gameRunning(g)
	if err != nil {
		return loader.Status{}, err
	}
	if running || (!fromStart && s.profiles.AnyRunning(id)) {
		return loader.Status{}, usererr.Wrap(usererr.Busy, fmt.Errorf("%s is running: close it before installing %s", g.Name(), game.LoaderName(id)))
	}
	return s.installVersion(ctx, g, dir, id, s.settings.Get().SmapiPin, fromStart)
}

// gameRunning reports whether any process of the game runs, with or without its loader and however it was started.
func (s *Service) gameRunning(g game.Game) (bool, error) {
	for _, name := range game.ProcessNames(g) {
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
