package bepinex5

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
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

const (
	packNamespace = "BepInEx"
	packName      = "BepInExPack"
	// lastReleases bounds the version list a picker shows.
	lastReleases = 10
)

// InProfile marks that BepInEx's files sit in each profile's folder, not in the game's install.
func (Loader) InProfile() {}

// thunderstoreDriver is the registered Thunderstore source, so the pack is read from the index the rest of Mortar
// shares; Index overrides it for tests.
func (l Loader) thunderstoreDriver() thunderstore.Driver {
	if l.Index != nil {
		return *l.Index
	}
	if e, ok := source.Get("thunderstore"); ok {
		if d, ok := e.Source.(thunderstore.Driver); ok {
			return d
		}
	}
	return thunderstore.Driver{}
}

func communityOf(g components.GameInfo) (string, error) {
	s, ok := g.Source("thunderstore")
	if !ok || s.Key == "" {
		return "", fmt.Errorf("%s has no Thunderstore community", g.Name)
	}
	return s.Key, nil
}

// Versions are the BepInEx 5 releases of BepInExPack in the game's community index, newest first.
func (l Loader) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	key, err := communityOf(g)
	if err != nil {
		return nil, err
	}
	all, err := l.thunderstoreDriver().Versions(ctx, key, packNamespace, packName, "")
	if err != nil {
		return nil, fmt.Errorf("look up BepInEx releases: %w", err)
	}
	var out []string
	for _, v := range all {
		if strings.HasPrefix(v, "5.") {
			out = append(out, v)
		}
		if len(out) == lastReleases {
			break
		}
	}
	return out, nil
}

// Latest is the newest BepInEx 5 release.
func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	all, err := l.Versions(ctx, g)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("the Thunderstore index lists no BepInEx 5 release")
	}
	return all[0], nil
}

// Fetch downloads the BepInExPack zip of version to dst.
func (l Loader) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	key, err := communityOf(g)
	if err != nil {
		return err
	}
	r, err := l.thunderstoreDriver().Resolve(ctx, key, packNamespace, packName, version, "")
	if err != nil {
		return fmt.Errorf("find BepInEx %s: %w", version, err)
	}
	work, err := os.MkdirTemp(filepath.Dir(dst), ".bepinex-fetch-*")
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(work) }()
	got, err := (&host.Direct{HTTP: l.HTTP}).Fetch(ctx, r.URL, work)
	if err != nil {
		return fmt.Errorf("download BepInEx %s: %w", version, err)
	}
	return fsx.Rename(got, dst)
}
