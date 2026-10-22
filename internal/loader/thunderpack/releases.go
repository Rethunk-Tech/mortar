package thunderpack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/host"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// lastReleases bounds the version list a picker shows.
const lastReleases = 10

// thunderstoreDriver is the registered Thunderstore source, so the pack is read from the index the rest of Mortar
// shares; Index overrides it for tests.
func (l Loader) thunderstoreDriver() thunderstore.Driver {
	if l.Index != nil {
		return *l.Index
	}
	d, _ := thunderstore.Registered()
	return d
}

// pack is the game's loader package: the Thunderstore community the game's mods come from, and the package's namespace
// and name.
func pack(g components.GameInfo) (community, namespace, name string, err error) {
	s, ok := g.Source("thunderstore")
	if !ok || s.Key == "" {
		return "", "", "", fmt.Errorf("%s has no Thunderstore community", g.Name)
	}
	namespace, name, ok = strings.Cut(g.LoaderPackage(), "-")
	if !ok {
		return "", "", "", fmt.Errorf("%s names no loader package", g.Name)
	}
	return s.Key, namespace, name, nil
}

// Versions are the releases of the game's loader package in its community index, newest first.
func (l Loader) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	key, namespace, name, err := pack(g)
	if err != nil {
		return nil, err
	}
	all, err := l.thunderstoreDriver().Versions(ctx, key, namespace, name, "")
	if err != nil {
		return nil, fmt.Errorf("look up %s releases: %w", l.Name, err)
	}
	return all[:min(len(all), lastReleases)], nil
}

// Latest is the package's newest release.
func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	all, err := l.Versions(ctx, g)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("the Thunderstore index lists no " + l.Name + " release")
	}
	return all[0], nil
}

// Fetch downloads the game's loader package zip of version to dst.
func (l Loader) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	key, namespace, name, err := pack(g)
	if err != nil {
		return err
	}
	r, err := l.thunderstoreDriver().Resolve(ctx, key, namespace, name, version, "")
	if err != nil {
		return fmt.Errorf("find %s %s: %w", l.Name, version, err)
	}
	work, err := os.MkdirTemp(filepath.Dir(dst), ".thunderpack-fetch-*")
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	got, err := (&host.Direct{HTTP: l.HTTP}).Fetch(ctx, r.URL, work)
	if err != nil {
		return fmt.Errorf("download %s %s: %w", l.Name, version, err)
	}
	return fsx.Rename(got, dst)
}
