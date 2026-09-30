package stardew

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

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/github"
)

const (
	releasesURL  = "https://api.github.com/repos/Pathoschild/SMAPI/releases?per_page=10"
	downloadBase = "https://github.com/Pathoschild/SMAPI/releases/download"
	cacheFile    = "smapi-release.json"
	cacheTTL     = time.Hour
	apiTimeout   = 8 * time.Second
	// maxInstaller bounds the download; the 4.5.2 installer is 42 MB.
	maxInstaller = 512 << 20
)

var versionPattern = regexp.MustCompile(`^\d+(\.\d+)*$`)

type cachedRelease struct {
	Fetched time.Time `json:"fetched"`
	Version string    `json:"version"`
}

func installerAsset(version string) string { return "SMAPI-" + version + "-installer.zip" }

func (g Game) cachePath() (string, error) {
	dir := g.CacheDir
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

// LatestLoader returns the newest stable SMAPI version that ships an installer. A response younger than an hour
// is reused, and a stale one stands in when the lookup fails.
func (g Game) LatestLoader(ctx context.Context) (string, error) {
	path, err := g.cachePath()
	if err != nil {
		return "", err
	}
	cached, ok := readCache(path)
	if ok && time.Since(cached.Fetched) < cacheTTL {
		return cached.Version, nil
	}
	version, err := g.fetchLatest(ctx)
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

func (g Game) fetchLatest(ctx context.Context) (string, error) {
	url := g.ReleasesURL
	if url == "" {
		url = releasesURL
	}
	all, err := github.FetchReleases(ctx, g.Client, url)
	if err != nil {
		return "", fmt.Errorf("look up SMAPI releases: %w", err)
	}
	for _, r := range all {
		v := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || r.Prerelease || !versionPattern.MatchString(v) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == installerAsset(v) {
				return v, nil
			}
		}
	}
	return "", errors.New("GitHub lists no stable SMAPI release with an installer")
}

// download saves the release installer to dest.
func (g Game) download(ctx context.Context, version, dest string) error {
	base := g.DownloadBase
	if base == "" {
		base = downloadBase
	}
	url := base + "/" + version + "/" + installerAsset(version)
	if err := github.Download(ctx, g.Client, url, dest, maxInstaller, nil); err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	return nil
}
