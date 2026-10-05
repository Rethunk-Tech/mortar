package smapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

const (
	cacheFile  = "smapi-release.json"
	cacheTTL   = time.Hour
	apiTimeout = 8 * time.Second
	// maxInstaller bounds the installer download.
	maxInstaller = 512 << 20
)

var versionPattern = regexp.MustCompile(`^\d+(\.\d+)*$`)

type cachedRelease struct {
	Fetched time.Time `json:"fetched"`
	Version string    `json:"version"`
}

func (l Loader) loaderComponent() (components.Component, bool) {
	client := l.Components
	if client == nil {
		client = configuredComponents.Load()
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

func (l Loader) cachePath() (string, error) {
	dir := l.CacheDir
	if dir == "" {
		base, err := datadir.Dir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "cache")
	}
	return filepath.Join(dir, cacheFile), nil
}

func readCache(path string) (cachedRelease, bool) {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return cachedRelease{}, false
	}
	var c cachedRelease
	if json.Unmarshal(b, &c) != nil || !versionPattern.MatchString(c.Version) {
		return cachedRelease{}, false
	}
	return c, true
}

func writeCache(path string, c cachedRelease) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, c)
}

// Latest returns the newest stable SMAPI version that ships an installer. A response younger than an hour
// is reused, and a stale one stands in when the lookup fails.
func (l Loader) Latest(ctx context.Context, _ components.GameInfo) (string, error) {
	if component, ok := l.loaderComponent(); ok {
		return component.Version, nil
	}
	path, err := l.cachePath()
	if err != nil {
		return "", err
	}
	cached, ok := readCache(path)
	if ok && time.Since(cached.Fetched) < cacheTTL {
		return cached.Version, nil
	}
	version, err := l.fetchLatest(ctx)
	if err != nil {
		if ok {
			return cached.Version, nil
		}
		return "", err
	}
	// A cache that cannot be written only costs a refetch.
	_ = writeCache(path, cachedRelease{Fetched: time.Now().UTC(), Version: version})
	return version, nil
}

func (l Loader) fetchLatest(ctx context.Context) (string, error) {
	url := l.ReleasesURL
	if url == "" {
		return "", errors.New("SMAPI component manifest is not configured")
	}
	all, err := github.FetchReleases(ctx, l.Client, url)
	if err != nil {
		return "", fmt.Errorf("look up SMAPI releases: %w", err)
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
	url := l.ReleasesURL
	if url == "" {
		if component, ok := l.loaderComponent(); ok && component.Source.Owner != "" && component.Source.Repo != "" {
			url = "https://api.github.com/repos/" + component.Source.Owner + "/" + component.Source.Repo + "/releases"
		}
	}
	if url == "" {
		return nil, errors.New("SMAPI component manifest is not configured")
	}
	all, err := github.FetchReleases(ctx, l.Client, url)
	if err != nil {
		return nil, fmt.Errorf("look up SMAPI releases: %w", err)
	}
	var out []string
	for _, r := range all {
		v := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || !versionPattern.MatchString(v) {
			continue
		}
		out = append(out, v)
		if len(out) == lastLoaderReleases {
			break
		}
	}
	return out, nil
}

// Fetch saves the release installer of version to dest.
func (l Loader) Fetch(ctx context.Context, _ components.GameInfo, version, dest string) error {
	if component, ok := l.loaderComponent(); ok && component.Version == version {
		client := l.Components
		if client == nil {
			client = configuredComponents.Load()
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
	url := base + "/" + version + "/" + asset
	if err := github.Download(ctx, l.Client, url, dest, maxInstaller, nil); err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	return nil
}
