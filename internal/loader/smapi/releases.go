package smapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
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

func (l Loader) cache() *meta.Client { return &meta.Client{CacheDir: l.CacheDir} }

// Latest returns the newest stable SMAPI version that ships an installer. A response younger than an hour
// is reused, and a stale one stands in when the lookup fails.
func (l Loader) Latest(ctx context.Context, _ components.GameInfo) (string, error) {
	if component, ok := l.loaderComponent(); ok {
		return component.Version, nil
	}
	return meta.Cached(l.cache(), cacheFile, cacheTTL, func() (string, error) { return l.fetchLatest(ctx) })
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
		if r.Draft || !versionPattern.MatchString(v) || !hasDigest(r, l.installerAsset(v)) {
			continue
		}
		out = append(out, v)
		if len(out) == lastLoaderReleases {
			break
		}
	}
	return out, nil
}

// hasDigest reports whether r lists asset with a SHA-256, which Fetch needs before it runs the installer. The older
// releases GitHub never hashed are left out of the choices rather than offered and refused.
func hasDigest(r github.Release, asset string) bool {
	return slices.ContainsFunc(r.Assets, func(a github.Asset) bool {
		return a.Name == asset && strings.HasPrefix(a.Digest, "sha256:") && len(a.Digest) > len("sha256:")
	})
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
	want, err := l.releaseDigest(ctx, version, asset)
	if err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	url := base + "/" + version + "/" + asset
	if err := github.Download(ctx, l.Client, url, dest, maxInstaller, nil); err != nil {
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	got, err := safefs.HashFile(dest, sha256.New())
	if err == nil && !strings.EqualFold(got, want) {
		err = errors.New("the download does not match GitHub's SHA-256 for it")
	}
	if err != nil {
		_ = safefs.Remove(dest)
		return fmt.Errorf("download SMAPI %s: %w", version, err)
	}
	return nil
}

// releaseDigest is the SHA-256 GitHub reports for version's installer asset. A release without one is refused,
// since the installer runs natively.
func (l Loader) releaseDigest(ctx context.Context, version, asset string) (string, error) {
	all, err := l.releases(ctx)
	if err != nil {
		return "", err
	}
	for _, r := range all {
		if r.Draft || strings.TrimPrefix(r.Tag, "v") != version {
			continue
		}
		for _, a := range r.Assets {
			if a.Name != asset {
				continue
			}
			if want, ok := strings.CutPrefix(a.Digest, "sha256:"); ok && want != "" {
				return want, nil
			}
			return "", errors.New("the release carries no SHA-256 to verify it by")
		}
	}
	return "", fmt.Errorf("GitHub lists no SMAPI release %s with that installer", version)
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
