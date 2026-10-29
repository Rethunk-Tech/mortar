package thunderpack

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/host"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// lastReleases bounds the version list a picker shows.
const lastReleases = 10

// Releases lists and downloads a game's loader package from its Thunderstore community index.
type Releases struct {
	// Name is the loader's name in errors.
	Name string
	// Index is the Thunderstore driver the pack is looked up in; nil means the registered one, so the pack is read
	// from the index the rest of Mortar shares.
	Index *thunderstore.Driver
	// HTTP downloads the pack.
	HTTP *http.Client
	// Keep picks the releases offered; nil keeps every one.
	Keep func(version string) bool
}

func (l Loader) releases() Releases { return Releases{Name: l.Name, Index: l.Index, HTTP: l.HTTP} }

func (r Releases) driver() thunderstore.Driver {
	if r.Index != nil {
		return *r.Index
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

// Versions are the kept releases of the game's loader package in its community index, newest first.
func (r Releases) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	key, namespace, name, err := pack(g)
	if err != nil {
		return nil, err
	}
	all, err := r.driver().Versions(ctx, key, namespace, name, "")
	if err != nil {
		return nil, fmt.Errorf("look up %s releases: %w", r.Name, err)
	}
	var out []string
	for _, v := range all {
		if len(out) == lastReleases {
			break
		}
		if r.Keep == nil || r.Keep(v) {
			out = append(out, v)
		}
	}
	return out, nil
}

// Latest is the package's newest kept release.
func (r Releases) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	all, err := r.Versions(ctx, g)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("the Thunderstore index lists no " + r.Name + " release")
	}
	return all[0], nil
}

// Fetch downloads the game's loader package zip of version to dst.
func (r Releases) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	key, namespace, name, err := pack(g)
	if err != nil {
		return err
	}
	res, err := r.driver().Resolve(ctx, key, namespace, name, version, "")
	if err != nil {
		return fmt.Errorf("find %s %s: %w", r.Name, version, err)
	}
	work, err := os.MkdirTemp(filepath.Dir(dst), ".thunderpack-fetch-*")
	if err != nil {
		return err
	}
	defer func() { _ = safefs.RemoveAll(work) }()
	got, err := (&host.Direct{HTTP: r.HTTP}).Fetch(ctx, res.URL, work)
	if err != nil {
		return fmt.Errorf("download %s %s: %w", r.Name, version, err)
	}
	return safefs.Rename(got, dst)
}

// Versions are the releases of the game's loader package in its community index, newest first.
func (l Loader) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	return l.releases().Versions(ctx, g)
}

// Latest is the package's newest release.
func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	return l.releases().Latest(ctx, g)
}

// Fetch downloads the game's loader package zip of version to dst.
func (l Loader) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	return l.releases().Fetch(ctx, g, version, dst)
}
