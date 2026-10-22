package bepinex5

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/host"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// lastReleases bounds the version list a picker shows.
const lastReleases = 10

// ProfileFiles are what InstallPack lays out in each profile's folder (BepInEx's files sit there, not in the game's
// install), with the marker that records the installed version.
func (Loader) ProfileFiles() []string {
	return append(append([]string{"BepInEx", "doorstop_config.ini", "doorstop_libs", doorstopFile, markerFile}, components.DoorstopProxies...), packRuntimeDirs...)
}

// thunderstoreDriver is the registered Thunderstore source, so the pack is read from the index the rest of Mortar
// shares; Index overrides it for tests.
func (l Loader) thunderstoreDriver() thunderstore.Driver {
	if l.Index != nil {
		return *l.Index
	}
	d, _ := thunderstore.Registered()
	return d
}

// pack is the game's BepInEx pack as a Thunderstore namespace and name.
func pack(g components.GameInfo) (namespace, name string) {
	namespace, name, _ = strings.Cut(g.LoaderPackage(), "-")
	return namespace, name
}

func communityOf(g components.GameInfo) (string, error) {
	s, ok := g.Source("thunderstore")
	if !ok || s.Key == "" {
		return "", fmt.Errorf("%s has no Thunderstore community", g.Name)
	}
	return s.Key, nil
}

// Versions are the releases of the game's BepInEx pack in its community index, newest first, of the BepInEx major the
// catalog names for the game (5 unless it says otherwise): plugins do not load across majors, and Mortar starts only
// BepInEx 5 on Mono and BepInEx 6 on IL2CPP.
func (l Loader) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	key, err := communityOf(g)
	if err != nil {
		return nil, err
	}
	namespace, name := pack(g)
	all, err := l.thunderstoreDriver().Versions(ctx, key, namespace, name, "")
	if err != nil {
		return nil, fmt.Errorf("look up BepInEx releases: %w", err)
	}
	major := strconv.Itoa(cmp.Or(g.LoaderMajor(ID), 5)) + "."
	var out []string
	for _, v := range all {
		if strings.HasPrefix(v, major) {
			out = append(out, v)
		}
		if len(out) == lastReleases {
			break
		}
	}
	return out, nil
}

// Latest is the pack's newest release.
func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	all, err := l.Versions(ctx, g)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("the Thunderstore index lists no BepInEx release")
	}
	return all[0], nil
}

// Fetch downloads the game's BepInEx pack zip of version to dst.
func (l Loader) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	key, err := communityOf(g)
	if err != nil {
		return err
	}
	namespace, name := pack(g)
	r, err := l.thunderstoreDriver().Resolve(ctx, key, namespace, name, version, "")
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
