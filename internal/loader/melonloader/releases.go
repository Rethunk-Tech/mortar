package melonloader

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

const (
	// The release is one catalog component shared by every MelonLoader game, so it is hashed once.
	componentGame = "melonloader"
	componentName = "loader"
	asset         = "MelonLoader.x64.zip"
	// listedReleases bounds the version list: older games need older loaders than the newest.
	listedReleases = 40
	// maxPack bounds a release download.
	maxPack = 256 << 20
	// maxChecksum bounds a checksum file.
	maxChecksum = 4 << 10
)

var versionPattern = regexp.MustCompile(`^\d+(\.\d+)*$`)

func (l Loader) component() (components.Component, error) {
	client := l.Components
	if client == nil {
		client = components.Active()
	}
	if client != nil {
		if c, ok := client.Component(componentGame, componentName); ok {
			return c, nil
		}
	}
	return components.Component{}, errors.New("the component catalog lists no MelonLoader release")
}

// Latest is the release the signed catalog names.
func (l Loader) Latest(_ context.Context, _ components.GameInfo) (string, error) {
	c, err := l.component()
	return c.Version, err
}

func (l Loader) releases(ctx context.Context, c components.Component) ([]github.Release, error) {
	base := l.APIBase
	if base == "" {
		base = "https://api.github.com"
	}
	return github.FetchReleases(ctx, l.HTTP, fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100", base, c.Source.Owner, c.Source.Repo))
}

// stable is the version a release is for, when it is a stable release that ships the Windows x64 loader.
func stable(r github.Release) (string, bool) {
	v := strings.TrimPrefix(r.Tag, "v")
	if r.Draft || r.Prerelease || !versionPattern.MatchString(v) {
		return "", false
	}
	for _, a := range r.Assets {
		if a.Name == asset {
			return v, true
		}
	}
	return "", false
}

// Versions are the stable releases GitHub lists, newest first.
func (l Loader) Versions(ctx context.Context, _ components.GameInfo) ([]string, error) {
	c, err := l.component()
	if err != nil {
		return nil, err
	}
	all, err := l.releases(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("look up MelonLoader releases: %w", err)
	}
	var out []string
	for _, r := range all {
		if v, ok := stable(r); ok {
			out = append(out, v)
		}
		if len(out) == listedReleases {
			break
		}
	}
	return out, nil
}

// Fetch downloads the Windows x64 loader of version to dst. The catalog's release is checked against the catalog's
// hash; any other is checked against the digest GitHub computed for the asset, or the .sha512 file the release holds
// beside it, and refused when it has neither.
func (l Loader) Fetch(ctx context.Context, _ components.GameInfo, version, dst string) error {
	c, err := l.component()
	if err != nil {
		return err
	}
	if c.Version == version {
		client := l.Components
		if client == nil {
			client = components.Active()
		}
		if err := client.Download(ctx, c, dst); err != nil {
			return fmt.Errorf("download MelonLoader %s: %w", version, err)
		}
		return nil
	}
	all, err := l.releases(ctx, c)
	if err != nil {
		return fmt.Errorf("look up MelonLoader releases: %w", err)
	}
	for _, r := range all {
		if v, ok := stable(r); !ok || v != version {
			continue
		}
		if err := l.download(ctx, r, dst); err != nil {
			return fmt.Errorf("download MelonLoader %s: %w", version, err)
		}
		return nil
	}
	return fmt.Errorf("GitHub lists no stable MelonLoader release %s", version)
}

func (l Loader) download(ctx context.Context, r github.Release, dst string) error {
	var pack github.Asset
	var sum string
	for _, a := range r.Assets {
		switch a.Name {
		case asset:
			pack = a
		case strings.TrimSuffix(asset, ".zip") + ".sha512":
			sum = a.URL
		}
	}
	if pack.Digest == "" && sum == "" {
		return errors.New("the release carries no checksum to verify it by")
	}
	if err := github.Download(ctx, l.HTTP, pack.URL, dst, maxPack, nil); err != nil {
		return err
	}
	if err := l.verify(ctx, pack, sum, dst); err != nil {
		_ = fsx.Remove(dst)
		return err
	}
	return nil
}

func (l Loader) verify(ctx context.Context, pack github.Asset, sumURL, file string) error {
	if want, ok := strings.CutPrefix(pack.Digest, "sha256:"); ok {
		got, err := fsx.SHA256(file)
		if err != nil {
			return err
		}
		if !strings.EqualFold(got, want) {
			return errors.New("the download does not match GitHub's SHA-256 for it")
		}
		return nil
	}
	if sumURL == "" {
		return errors.New("GitHub's digest for the release is not a SHA-256")
	}
	want, err := l.checksum(ctx, sumURL)
	if err != nil {
		return err
	}
	f, err := fsx.Open(file)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), want) {
		return errors.New("the download does not match the release's SHA-512 file")
	}
	return nil
}

// checksum is the hex digest a .sha512 file holds, which may be followed by the file's name.
func (l Loader) checksum(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	hc := l.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the checksum file answered %s", resp.Status)
	}
	b, err := fsx.ReadCapped(resp.Body, maxChecksum)
	if err != nil {
		return "", err
	}
	if fields := strings.Fields(string(b)); len(fields) > 0 {
		return fields[0], nil
	}
	return "", errors.New("the checksum file is empty")
}
