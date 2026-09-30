package stardew

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/datadir"
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

type release struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

type cachedRelease struct {
	Fetched time.Time `json:"fetched"`
	Version string    `json:"version"`
}

func installerAsset(version string) string { return "SMAPI-" + version + "-installer.zip" }

func (g Game) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

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
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach GitHub to look up SMAPI releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := rateLimited(resp); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub's releases API answered %s", resp.Status)
	}
	var all []release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&all); err != nil {
		return "", fmt.Errorf("read GitHub's releases response: %w", err)
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

func rateLimited(resp *http.Response) error {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}
	if resp.Header.Get("X-Ratelimit-Remaining") != "0" && resp.Header.Get("Retry-After") == "" {
		return nil
	}
	msg := "GitHub's rate limit for unauthenticated requests is used up"
	if secs, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64); err == nil {
		msg += "; try again after " + time.Unix(secs, 0).Format("15:04")
	}
	return errors.New(msg)
}

// download saves the release installer to dest.
func (g Game) download(ctx context.Context, version, dest string) error {
	base := g.DownloadBase
	if base == "" {
		base = downloadBase
	}
	url := base + "/" + version + "/" + installerAsset(version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := rateLimited(resp); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download SMAPI %s: server answered %s", version, resp.Status)
	}
	out, err := fsx.Create(dest)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(resp.Body, maxInstaller+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	if n > maxInstaller {
		return fmt.Errorf("download SMAPI %s: larger than %d MiB", version, maxInstaller>>20)
	}
	return nil
}
