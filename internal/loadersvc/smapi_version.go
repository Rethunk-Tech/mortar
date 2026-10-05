package loadersvc

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var errUnknownSMAPI = errors.New("unknown SMAPI version")

// InstallVersion installs the given SMAPI version. A copy already in the store is applied without downloading.
func (s *Service) InstallVersion(ctx context.Context, id, version string) (loader.Status, error) {
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	// A version that does not exist is refused before asking the user to close the game.
	if _, err := s.items.Path(id, store.SMAPIKey(version)); version != "" && errors.Is(err, store.ErrNotFound) {
		if err := s.ensureKnown(ctx, id, version); err != nil {
			return loader.Status{}, err
		}
	}
	running, err := s.gameRunning(g)
	if err != nil {
		return loader.Status{}, err
	}
	if running || s.profiles.AnyRunning(id) {
		return loader.Status{}, usererr.Wrap(usererr.Busy, fmt.Errorf("%s is running: close it before installing %s", g.Name(), game.LoaderName(id)))
	}
	return s.installVersion(ctx, g, dir, id, version, false)
}

// ListVersions is store-held SMAPI versions plus the last GitHub releases, newest first, unique.
func (s *Service) ListVersions(ctx context.Context, id string) ([]string, error) {
	if _, err := game.Require(id); err != nil {
		return nil, err
	}
	remote, err := s.remoteVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range remote {
		add(v)
	}
	for _, v := range s.storedVersions(id) {
		add(v)
	}
	return out, nil
}

func (s *Service) installVersion(ctx context.Context, g game.Game, dir, id, version string, fromStart bool) (st loader.Status, err error) {
	s.emit(StateEvent, State{Game: id, Installing: true})
	defer func() {
		state := State{Game: id}
		if err != nil {
			state.Error = err.Error()
		}
		s.emit(StateEvent, state)
	}()
	version, err = s.resolveVersion(ctx, id, g, version)
	if err != nil {
		return loader.Status{}, err
	}
	if _, err := s.items.Path(id, store.SMAPIKey(version)); err == nil {
		if err := s.recordLoader(id, version, fromStart); err != nil {
			return loader.Status{}, err
		}
		return s.Status(ctx, id)
	} else if !errors.Is(err, store.ErrNotFound) {
		return loader.Status{}, err
	}
	if err := s.ensureKnown(ctx, id, version); err != nil {
		return loader.Status{}, err
	}
	if s.fetchInstall != nil {
		if err := s.fetchInstall(ctx, id, version); err != nil {
			return loader.Status{}, err
		}
		if err := s.recordLoader(id, version, fromStart); err != nil {
			return loader.Status{}, err
		}
		return s.Status(ctx, id)
	}
	l, ok := game.PrimaryLoader(id)
	rel, canFetch := l.(loader.Releases)
	if !ok || !canFetch {
		return loader.Status{}, fmt.Errorf("%s does not install a chosen SMAPI version", g.Name())
	}
	bundled := func(got, modsDir string) error {
		key := store.SMAPIKey(got)
		if err := s.items.AddDir(id, key, modsDir); err != nil {
			return err
		}
		return s.applyBundled(id, key, fromStart)
	}
	work, err := os.MkdirTemp("", "mortar-loader-")
	if err != nil {
		return loader.Status{}, err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	pkg := loader.Package{ID: l.ID(), Version: version, Archive: filepath.Join(work, "installer.zip")}
	if err := rel.Fetch(ctx, version, pkg.Archive); err != nil {
		return loader.Status{}, err
	}
	progress := func(step loader.Step) { s.emit(ProgressEvent, Progress{Game: id, Step: step}) }
	progress(loader.StepDownloaded)
	got, err := l.Install(ctx, loader.Target{Game: id, InstallDir: dir, Bundled: bundled}, pkg, progress)
	if err != nil {
		return loader.Status{}, err
	}
	if err := s.recordLoader(id, got, fromStart); err != nil {
		return loader.Status{}, err
	}
	return s.Status(ctx, id)
}

func (s *Service) resolveVersion(ctx context.Context, id string, g game.Game, version string) (string, error) {
	if version != "" {
		return version, nil
	}
	if s.listVersions != nil {
		all, err := s.listVersions(ctx, id)
		if err != nil {
			return "", err
		}
		if len(all) == 0 {
			return "", errUnknownSMAPI
		}
		return all[0], nil
	}
	l, _ := game.PrimaryLoader(id)
	rel, ok := l.(loader.Releases)
	if !ok {
		return "", fmt.Errorf("%s does not list SMAPI versions", g.Name())
	}
	return rel.Latest(ctx)
}

func (s *Service) ensureKnown(ctx context.Context, id, version string) error {
	all, err := s.remoteVersions(ctx, id)
	if err != nil {
		return err
	}
	if slices.Contains(all, version) {
		return nil
	}
	return fmt.Errorf("%w: %s", errUnknownSMAPI, version)
}

func (s *Service) remoteVersions(ctx context.Context, id string) ([]string, error) {
	if s.listVersions != nil {
		return s.listVersions(ctx, id)
	}
	g, err := game.Require(id)
	if err != nil {
		return nil, err
	}
	l, _ := game.PrimaryLoader(id)
	rel, ok := l.(loader.Releases)
	if !ok {
		return nil, fmt.Errorf("%s does not list SMAPI versions", g.Name())
	}
	return rel.Versions(ctx)
}

func (s *Service) storedVersions(id string) []string {
	keys, err := s.items.Keys(id)
	if err != nil {
		return nil
	}
	var out []string
	for _, key := range keys {
		if v, ok := store.SMAPIVersion(key); ok {
			out = append(out, v)
		}
	}
	return out
}

func (s *Service) applyBundled(id, key string, fromStart bool) error {
	apply := s.profiles.ApplyBundled
	if fromStart {
		apply = s.profiles.ApplyBundledForStart
	}
	return apply(id, profile.Bundle{Key: key, Source: profile.Source{Kind: profile.SourceSMAPI, Name: "SMAPI"}})
}

func (s *Service) recordLoader(id, version string, fromStart bool) error {
	if _, err := s.items.Path(id, store.SMAPIKey(version)); err == nil {
		if err := s.applyBundled(id, store.SMAPIKey(version), fromStart); err != nil {
			return err
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	next, err := s.settings.Update(func(v *settings.Settings) {
		v.Loaders = maps.Clone(v.Loaders)
		v.Loaders[id] = version
	})
	if err != nil {
		return err
	}
	s.emit(settings.ChangedEvent, next)
	if s.OnReady != nil {
		s.OnReady(id)
	}
	return nil
}
