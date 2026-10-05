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

var errUnknownVersion = errors.New("unknown loader version")

// InstallVersion installs the given SMAPI version. A copy already in the store is applied without downloading.
func (s *Service) InstallVersion(ctx context.Context, id, loaderID, version string) (loader.Status, error) {
	if !s.busy.TryLock() {
		return loader.Status{}, errors.New("a loader install is already running")
	}
	defer s.busy.Unlock()
	g, dir, err := s.target(id)
	if err != nil {
		return loader.Status{}, err
	}
	l, ok := game.LoaderOf(id, loaderID)
	if !ok {
		return loader.Status{}, fmt.Errorf("%s has no loader", g.Name())
	}
	// A version that does not exist is refused before asking the user to close the game.
	if _, err := s.items.Path(id, store.LoaderKey(l.ID(), version)); version != "" && errors.Is(err, store.ErrNotFound) {
		if err := s.ensureKnown(ctx, id, loaderID, version); err != nil {
			return loader.Status{}, err
		}
	}
	running, err := s.gameRunning(g)
	if err != nil {
		return loader.Status{}, err
	}
	if running || s.profiles.AnyRunning(id) {
		return loader.Status{}, usererr.Wrap(usererr.Busy, fmt.Errorf("%s is running: close it before installing %s", g.Name(), game.LoaderName(id, loaderID)))
	}
	return s.installVersion(ctx, g, dir, id, loaderID, version, false)
}

// ListVersions is store-held SMAPI versions plus the last GitHub releases, newest first, unique.
func (s *Service) ListVersions(ctx context.Context, id, loaderID string) ([]string, error) {
	if _, err := game.Require(id); err != nil {
		return nil, err
	}
	remote, err := s.remoteVersions(ctx, id, loaderID)
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
	for _, v := range s.storedVersions(id, loaderID) {
		add(v)
	}
	return out, nil
}

func (s *Service) installVersion(ctx context.Context, g game.Game, dir, id, loaderID, version string, fromStart bool) (st loader.Status, err error) {
	s.emit(StateEvent, State{Game: id, Installing: true})
	defer func() {
		state := State{Game: id}
		if err != nil {
			state.Error = err.Error()
		}
		s.emit(StateEvent, state)
	}()
	l, ok := game.LoaderOf(id, loaderID)
	if !ok {
		return loader.Status{}, fmt.Errorf("%s has no loader", g.Name())
	}
	_, perProfile := l.(loader.InProfile)
	if version == "" && perProfile {
		version = s.settings.Get().Loaders[l.ID()]
	}
	version, err = s.resolveVersion(ctx, id, loaderID, g, version)
	if err != nil {
		return loader.Status{}, err
	}
	if perProfile {
		return s.installInProfiles(ctx, id, loaderID, dir, l, version, fromStart)
	}
	if _, err := s.items.Path(id, store.LoaderKey(l.ID(), version)); err == nil {
		if err := s.recordLoader(id, l, version, fromStart); err != nil {
			return loader.Status{}, err
		}
		return s.Status(ctx, id, loaderID)
	} else if !errors.Is(err, store.ErrNotFound) {
		return loader.Status{}, err
	}
	if err := s.ensureKnown(ctx, id, loaderID, version); err != nil {
		return loader.Status{}, err
	}
	if s.fetchInstall != nil {
		if err := s.fetchInstall(ctx, id, version); err != nil {
			return loader.Status{}, err
		}
		if err := s.recordLoader(id, l, version, fromStart); err != nil {
			return loader.Status{}, err
		}
		return s.Status(ctx, id, loaderID)
	}
	rel, canFetch := l.(loader.Releases)
	if !canFetch {
		return loader.Status{}, fmt.Errorf("%s does not install a chosen loader version", g.Name())
	}
	bundled := func(got, modsDir string) error {
		key := store.LoaderKey(l.ID(), got)
		if err := s.items.AddDir(id, key, modsDir); err != nil {
			return err
		}
		return s.applyBundled(id, l, key, fromStart)
	}
	work, err := os.MkdirTemp("", "mortar-loader-")
	if err != nil {
		return loader.Status{}, err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	pkg := loader.Package{ID: l.ID(), Version: version, Archive: filepath.Join(work, "installer.zip")}
	if err := rel.Fetch(ctx, catalogGame(id), version, pkg.Archive); err != nil {
		return loader.Status{}, err
	}
	progress := func(step loader.Step) { s.emit(ProgressEvent, Progress{Game: id, Step: step}) }
	progress(loader.StepDownloaded)
	got, err := l.Install(ctx, loader.Target{Game: id, InstallDir: dir, Exec: s.runtimeExec(id), Bundled: bundled}, pkg, progress)
	if err != nil {
		return loader.Status{}, err
	}
	if err := s.recordLoader(id, l, got, fromStart); err != nil {
		return loader.Status{}, err
	}
	return s.Status(ctx, id, loaderID)
}

func (s *Service) resolveVersion(ctx context.Context, id, loaderID string, g game.Game, version string) (string, error) {
	if version != "" {
		return version, nil
	}
	if s.listVersions != nil {
		all, err := s.listVersions(ctx, id)
		if err != nil {
			return "", err
		}
		if len(all) == 0 {
			return "", errUnknownVersion
		}
		return all[0], nil
	}
	l, _ := game.LoaderOf(id, loaderID)
	rel, ok := l.(loader.Releases)
	if !ok {
		return "", fmt.Errorf("%s does not list loader versions", g.Name())
	}
	return rel.Latest(ctx, catalogGame(id))
}

func (s *Service) ensureKnown(ctx context.Context, id, loaderID, version string) error {
	all, err := s.remoteVersions(ctx, id, loaderID)
	if err != nil {
		return err
	}
	if slices.Contains(all, version) {
		return nil
	}
	return fmt.Errorf("%w: %s", errUnknownVersion, version)
}

func (s *Service) remoteVersions(ctx context.Context, id, loaderID string) ([]string, error) {
	if s.listVersions != nil {
		return s.listVersions(ctx, id)
	}
	g, err := game.Require(id)
	if err != nil {
		return nil, err
	}
	l, _ := game.LoaderOf(id, loaderID)
	rel, ok := l.(loader.Releases)
	if !ok {
		return nil, fmt.Errorf("%s does not list loader versions", g.Name())
	}
	return rel.Versions(ctx, catalogGame(id))
}

func (s *Service) storedVersions(id, loaderID string) []string {
	l, ok := game.LoaderOf(id, loaderID)
	if !ok {
		return nil
	}
	want := l.ID()
	keys, err := s.items.Keys(id)
	if err != nil {
		return nil
	}
	var out []string
	for _, key := range keys {
		if lid, v, ok := store.LoaderOf(key); ok && lid == want {
			out = append(out, v)
		}
	}
	return out
}

func (s *Service) applyBundled(id string, l loader.Loader, key string, fromStart bool) error {
	b, ok := bundleOf(l, key)
	if !ok {
		return nil
	}
	apply := s.profiles.ApplyBundled
	if fromStart {
		apply = s.profiles.ApplyBundledForStart
	}
	return apply(id, b)
}

// bundleOf is the profile entry for the loader's bundled mods under key; ok is false for a loader that ships none.
func bundleOf(l loader.Loader, key string) (profile.Bundle, bool) {
	c, ok := l.(loader.BundledCopier)
	if !ok {
		return profile.Bundle{}, false
	}
	kind, name := c.BundleSource()
	return profile.Bundle{Key: key, Source: profile.Source{Kind: kind, Name: name}}, true
}

func (s *Service) recordLoader(id string, l loader.Loader, version string, fromStart bool) error {
	if _, err := s.items.Path(id, store.LoaderKey(l.ID(), version)); err == nil {
		if err := s.applyBundled(id, l, store.LoaderKey(l.ID(), version), fromStart); err != nil {
			return err
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	next, err := s.settings.Update(func(v *settings.Settings) {
		v.Loaders = maps.Clone(v.Loaders)
		v.Loaders[l.ID()] = version
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

// installInProfiles keeps the loader's installer in the store, so a later profile needs no download, and lays it into
// every profile of the game.
func (s *Service) installInProfiles(ctx context.Context, id, loaderID, dir string, l loader.Loader, version string, fromStart bool) (loader.Status, error) {
	rel, ok := l.(loader.Releases)
	if !ok {
		return loader.Status{}, fmt.Errorf("%s does not install a chosen loader version", l.ID())
	}
	key := store.LoaderKey(l.ID(), version)
	held, err := s.items.Path(id, key)
	if errors.Is(err, store.ErrNotFound) {
		if held, err = s.fetchIntoStore(ctx, id, l.ID(), rel, key, version); err != nil {
			return loader.Status{}, err
		}
	} else if err != nil {
		return loader.Status{}, err
	}
	all, err := s.profiles.List(id)
	if err != nil {
		return loader.Status{}, err
	}
	progress := func(step loader.Step) { s.emit(ProgressEvent, Progress{Game: id, Step: step}) }
	for _, p := range all {
		if p.Error != "" || s.profiles.LoaderID(id, p.ID) != l.ID() {
			continue
		}
		pdir, err := s.profiles.ProfileDir(id, p.ID)
		if err != nil {
			return loader.Status{}, err
		}
		pkg := loader.Package{ID: l.ID(), Version: version, Archive: filepath.Join(held, installerFile)}
		if _, err := l.Install(ctx, loader.Target{Game: id, InstallDir: dir, ProfileDir: pdir, Exec: s.runtimeExec(id)}, pkg, progress); err != nil {
			return loader.Status{}, fmt.Errorf("install into %s: %w", p.Name, err)
		}
	}
	if err := s.recordLoader(id, l, version, fromStart); err != nil {
		return loader.Status{}, err
	}
	return s.Status(ctx, id, loaderID)
}

// installerFile names the downloaded installer inside a per-profile loader's store entry.
const installerFile = "installer.zip"

// fetchIntoStore downloads the loader's installer into the store under key and returns its folder.
func (s *Service) fetchIntoStore(ctx context.Context, id, loaderID string, rel loader.Releases, key, version string) (string, error) {
	if err := s.ensureKnown(ctx, id, loaderID, version); err != nil {
		return "", err
	}
	work, err := os.MkdirTemp("", "mortar-loader-")
	if err != nil {
		return "", err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	if err := rel.Fetch(ctx, catalogGame(id), version, filepath.Join(work, installerFile)); err != nil {
		return "", err
	}
	s.emit(ProgressEvent, Progress{Game: id, Step: loader.StepDownloaded})
	if err := s.items.AddDir(id, key, work); err != nil {
		return "", err
	}
	return s.items.Path(id, key)
}
