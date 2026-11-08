package smapi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	cacheFile  = "smapi-release.json"
	cacheTTL   = time.Hour
	apiTimeout = 8 * time.Second
	// maxInstaller bounds the installer download.
	maxInstaller = 512 << 20
)

var versionPattern = regexp.MustCompile(`^\d+(\.\d+)*$`)

func (l Loader) loaderComponent() (components.Component, bool) {
	client := l.Components
	if client == nil {
		client = components.Active()
	}
	if client == nil {
		return components.Component{}, false
	}
	return client.Component(gameID, "smapi")
}

func (l Loader) installerAsset(version string) string {
	if component, ok := l.loaderComponent(); ok {
		if component.Version == version {
			return component.Asset
		}
		if component.Version != "" {
			return strings.ReplaceAll(component.Asset, component.Version, version)
		}
	}
	pattern := l.AssetPattern
	if pattern == "" {
		return ""
	}
	return strings.ReplaceAll(pattern, "{version}", version)
}

// cache is CacheDir, else <datadir>/cache, as every meta lookup is: the registered Loader sets no folder.
func (l Loader) cache() *diskcache.Cache { return (&meta.Client{CacheDir: l.CacheDir}).Cache() }

// Latest returns the newest stable SMAPI version that ships an installer. A response younger than an hour
// is reused, and a stale one stands in when the lookup fails.
func (l Loader) Latest(ctx context.Context, _ components.GameInfo) (string, error) {
	if component, ok := l.loaderComponent(); ok {
		return component.Version, nil
	}
	return diskcache.Cached(l.cache(), cacheFile, cacheTTL, func() (string, error) { return l.fetchLatest(ctx) })
}

func (l Loader) fetchLatest(ctx context.Context) (string, error) {
	all, err := l.releases(ctx)
	if err != nil {
		return "", err
	}
	for _, r := range all {
		v := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || r.Prerelease || !versionPattern.MatchString(v) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == l.installerAsset(v) {
				return v, nil
			}
		}
	}
	return "", errors.New("GitHub lists no stable SMAPI release with an installer")
}

const lastLoaderReleases = 10

// Versions returns the last GitHub SMAPI releases that look like versions, newest first.
func (l Loader) Versions(ctx context.Context, _ components.GameInfo) ([]string, error) {
	all, err := l.releases(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range all {
		v := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || !versionPattern.MatchString(v) || !l.hasPin(v) {
			continue
		}
		out = append(out, v)
		if len(out) == lastLoaderReleases {
			break
		}
	}
	return out, nil
}

func (l Loader) hasPin(version string) bool {
	_, ok := l.pinned(version)
	return ok
}

// pinned is the SHA-256 the signed catalog vouches for SMAPI version's installer.
func (l Loader) pinned(version string) (string, bool) {
	client := l.Components
	if client == nil {
		client = components.Active()
	}
	if client == nil {
		return "", false
	}
	return client.Pin(components.PinComponent, gameID, "smapi", version)
}

// Fetch saves the release installer of version to dest.
func (l Loader) Fetch(ctx context.Context, _ components.GameInfo, version, dest string) error {
	if component, ok := l.loaderComponent(); ok && component.Version == version {
		client := l.Components
		if client == nil {
			client = components.Active()
		}
		if err := client.Download(ctx, component, dest); err != nil {
			return fmt.Errorf("download SMAPI %s: %w", version, err)
		}
		return nil
	}
	base := l.DownloadBase
	if base == "" {
		if component, ok := l.loaderComponent(); ok && component.Source.Owner != "" && component.Source.Repo != "" {
			base = "https://github.com/" + component.Source.Owner + "/" + component.Source.Repo + "/releases/download"
		}
	}
	if base == "" {
		return errors.New("SMAPI component manifest is not configured")
	}
	asset := l.installerAsset(version)
	if asset == "" {
		return errors.New("SMAPI component asset pattern is not configured")
	}
	// The installer runs natively, and GitHub's own digest for an asset says only that the file is the one its
	// publisher uploaded, so a version the signed catalog does not pin is not installed.
	want, ok := l.pinned(version)
	if !ok {
		return usererr.New(usererr.Invalid, fmt.Sprintf("SMAPI %s is not in Mortar's signed catalog, so Mortar will not run its installer.", version))
	}
	url := base + "/" + version + "/" + asset
	err := github.Download(ctx, l.Client, url, dest, maxInstaller, "sha256:"+strings.ToLower(want), nil)
	if errors.Is(err, download.ErrChecksum) {
		err = errors.New("the download does not match the checksum in Mortar's signed catalog")
	}
	if err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	return nil
}

// releases are SMAPI's GitHub releases.
func (l Loader) releases(ctx context.Context) ([]github.Release, error) {
	url := l.releasesURL()
	if url == "" {
		return nil, errors.New("SMAPI component manifest is not configured")
	}
	all, err := github.FetchReleases(ctx, l.Client, url)
	if err != nil {
		return nil, fmt.Errorf("look up SMAPI releases: %w", err)
	}
	return all, nil
}

func (l Loader) releasesURL() string {
	if l.ReleasesURL != "" {
		return l.ReleasesURL
	}
	if component, ok := l.loaderComponent(); ok && component.Source.Owner != "" && component.Source.Repo != "" {
		return "https://api.github.com/repos/" + component.Source.Owner + "/" + component.Source.Repo + "/releases"
	}
	return ""
}
