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
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
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
	run func(ctx context.Context, id, loaderID string, fromStart bool) (loader.Status, error)
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
	s.profiles.Bundled = func(id string) []profile.Bundle { return s.bundles(context.Background(), id) }
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
func (s *Service) bundles(ctx context.Context, id string) []profile.Bundle {
	var out []profile.Bundle
	if key, err := s.ensureBundled(ctx, id); err != nil {
		log.Printf("bundled mods for %s: %v", id, err)
	} else if key != "" {
		l, _ := game.PrimaryLoader(id)
		if b, ok := bundleOf(l, key); ok {
			out = append(out, b)
		}
	}
	if b, err := s.ensureBridge(ctx, id); err != nil {
		log.Printf("bridge for %s: %v", id, err)
	} else if b.Key != "" {
		out = append(out, b)
	}
	return out
}

// SyncBundled makes sure the game's bundled mods are in the store and in every profile. It never fails the
// caller: the loader may be absent or the game not installed.
func SyncBundled(ctx context.Context, s *Service, id string) {
	for _, b := range s.bundles(ctx, id) {
		if err := s.profiles.ApplyBundled(id, b); err != nil {
			log.Printf("bundled mods for %s: %v", id, err)
		}
	}
}

// ensureBridge returns the manifest's console bridge entry, adding it to the store first when it is missing.
// The bridge needs neither the loader nor the game folder, so every profile has it from creation.
func (s *Service) ensureBridge(ctx context.Context, id string) (profile.Bundle, error) {
	component, localZip, local := localBridge(id)
	if !local {
		if s.components == nil {
			return profile.Bundle{}, nil
		}
		var ok bool
		if component, ok = s.components.Component(id, "bridge"); !ok || component.Kind != "bridge" {
			return profile.Bundle{}, nil
		}
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
	if local {
		archivePath = localZip
	} else if err := s.components.Download(ctx, component, archivePath); err != nil {
		return profile.Bundle{}, err
	}
	unpacked := filepath.Join(tmp, "unpacked")
	if err := os.Mkdir(unpacked, 0o700); err != nil {
		return profile.Bundle{}, err
	}
	if err := archive.Extract(archivePath, unpacked); err != nil {
		return profile.Bundle{}, err
	}
	return b, s.items.AddDir(ctx, id, key, unpacked)
}

// ensureInBackground installs the game's loader when it is missing or broken, without blocking the caller.
// Failures are logged and announced with StateEvent; the game may simply not be installed.
func (s *Service) ensureInBackground(id string) {
	s.background.Go(func() {
		if _, err := s.Ensure(context.Background(), id, "", false); err != nil {
			log.Printf("loader for %s: %v", id, err)
		}
	})
}

// ensureBundled returns the store key of the installed loader's bundled mods, building the entry from the game
// folder when the loader was installed outside Mortar. It returns "" when no loader is installed.
func (s *Service) ensureBundled(ctx context.Context, id string) (string, error) {
	st, err := s.LocalStatus(id, "")
	if err != nil {
		return "", err
	}
	if !st.Installed || st.Broken || st.Version == "" {
		return "", nil
	}
	l, ok := game.PrimaryLoader(id)
	if _, ships := l.(loader.BundledCopier); !ok || !ships {
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
	if err := s.items.AddDir(ctx, id, key, tmp); err != nil {
		return "", err
	}
	if s.settings.Get().Loaders[settings.LoaderKey(id, l.ID())] == "" {
		if _, err := s.settings.Update(func(v *settings.Settings) {
			v.Loaders = maps.Clone(v.Loaders)
			v.Loaders[settings.LoaderKey(id, l.ID())] = st.Version
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

// LocalStatus reports the state on disk of the game's loader (loaderID, "" for the primary) without any network call.
func (s *Service) LocalStatus(id, loaderID string) (loader.Status, error) {
	_, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	if l, ok := game.LoaderOf(id, loaderID); ok {
		if _, perProfile := l.(loader.InProfile); perProfile {
			return s.profileStatus(id, l, dir, s.settings.Get().Loaders[settings.LoaderKey(id, l.ID())])
		}
	}
	return game.LoaderStatus(id, loaderID, dir, s.settings.Get().Loaders)
}

// profileStatus is the state of a loader that lives in each profile: installed when a version is recorded and every
// profile holds it.
func (s *Service) profileStatus(id string, l loader.Loader, dir, recorded string) (loader.Status, error) {
	all, err := s.profiles.List(id)
	if err != nil {
		return loader.Status{}, err
	}
	st := loader.Status{Installed: recorded != "", Version: recorded, PerProfile: true}
	for _, p := range all {
		if p.Error != "" || p.LoaderFor(id) != l.ID() {
			continue
		}
		pdir, err := s.profiles.ProfileDir(id, p.ID)
		if err != nil {
			return loader.Status{}, err
		}
		got, err := l.Status(loader.Target{Game: id, InstallDir: dir, ProfileDir: pdir})
		if err != nil {
			return loader.Status{}, err
		}
		if !got.Installed || got.Version != recorded {
			st.Installed = false
		}
	}
	return st, nil
}

// Status is LocalStatus plus whether a newer release exists. A failed release lookup only leaves Latest empty.
func (s *Service) Status(ctx context.Context, id, loaderID string) (loader.Status, error) {
	if _, _, err := s.target(id); err != nil {
		return loader.Status{}, err
	}
	st, err := s.LocalStatus(id, loaderID)
	if err != nil {
		return loader.Status{}, err
	}
	l, _ := game.LoaderOf(id, loaderID)
	if l == nil {
		return st, nil
	}
	if pin := s.settings.Get().LoaderPin(id, l.ID()); pin != "" {
		return st, nil
	}
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
func (s *Service) Install(ctx context.Context, id, loaderID string) (loader.Status, error) {
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	return s.run(ctx, id, loaderID, false)
}

// Ensure installs the loader when it is missing or broken, waiting for an install already running, and does
// nothing when it is fine. A newer release is never applied here: an update can break mods, so the user does it.
// fromStart is true when Play requested this install, so a preparing claim for that Start is not treated as running.
//
//wails:ignore
func (s *Service) Ensure(ctx context.Context, id, loaderID string, fromStart bool) (loader.Status, error) {
	s.busy.Lock()
	defer s.busy.Unlock()
	st, err := s.LocalStatus(id, loaderID)
	if err != nil {
		return loader.Status{}, err
	}
	pin := s.settings.Get().LoaderPin(id, s.loaderID(id, loaderID))
	if st.Installed && !st.Broken && (pin == "" || st.Version == pin) {
		return st, nil
	}
	return s.run(ctx, id, loaderID, fromStart)
}

// install runs one install with busy held and announces its start and end.
func (s *Service) install(ctx context.Context, id, loaderID string, fromStart bool) (st loader.Status, err error) {
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	running, err := s.gameRunning(g)
	if err != nil {
		return loader.Status{}, err
	}
	if running || (!fromStart && s.profiles.AnyRunning(id)) {
		return loader.Status{}, usererr.Wrap(usererr.Busy, fmt.Errorf("%s is running: close it before installing %s", g.Name(), game.LoaderName(id, loaderID)))
	}
	return s.installVersion(ctx, g, dir, id, loaderID, s.settings.Get().LoaderPin(id, s.loaderID(id, loaderID)), fromStart)
}

// gameRunning reports whether any process of the game runs, with or without its loader and however it was started.
func (s *Service) gameRunning(g game.Game) (bool, error) {
	procs, err := launch.Processes(s.procDir, game.ProcessNames(g)...)
	if err != nil {
		return false, fmt.Errorf("check for a running %s: %w", g.Name(), err)
	}
	return len(procs) > 0, nil
}

func (s *Service) emit(name string, data any) {
	if s.App != nil {
		s.App.Event.Emit(name, data)
	}
}

// loaderID is the id of the game's loader (loaderID, "" for the primary), "" when it has none.
func (s *Service) loaderID(gameID, loaderID string) string {
	if l, ok := game.LoaderOf(gameID, loaderID); ok {
		return l.ID()
	}
	return ""
}

// runtimeExec runs a program through the selected install's runtime, for an installer the host cannot run itself (a
// Windows one in a Bottles bottle); nil when the install runs natively.
func (s *Service) runtimeExec(id string) func(context.Context, launchplan.RuntimeReq, []string) error {
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), id, "")
	if err != nil || !inst.RunsInBottle() {
		return nil
	}
	return func(ctx context.Context, _ launchplan.RuntimeReq, argv []string) error {
		return inst.RunInBottle(ctx, runtime.ExecRunner, argv[0], argv[1:]...)
	}
}
