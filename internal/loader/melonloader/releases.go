package melonloader

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
		if v, ok := stable(r); ok && l.hasPin(v) {
			out = append(out, v)
		}
		if len(out) == listedReleases {
			break
		}
	}
	return out, nil
}

func (l Loader) client() *components.Client {
	if l.Components != nil {
		return l.Components
	}
	return components.Active()
}

func (l Loader) hasPin(version string) bool {
	c := l.client()
	if c == nil {
		return false
	}
	_, ok := c.Pin(components.PinComponent, componentGame, componentName, version)
	return ok
}

// Fetch downloads the Windows x64 loader of version to dst, checked against the SHA-256 the signed catalog pins for
// that version. The loader is injected into the game, and GitHub's own digest for an asset says only that the file is
// the one its publisher uploaded, so a version the catalog does not pin is not installed.
func (l Loader) Fetch(ctx context.Context, _ components.GameInfo, version, dst string) error {
	c, err := l.component()
	if err != nil {
		return err
	}
	client := l.client()
	if c.Version == version {
		if err := client.Download(ctx, c, dst); err != nil {
			return fmt.Errorf("download MelonLoader %s: %w", version, err)
		}
		return nil
	}
	want, ok := client.Pin(components.PinComponent, componentGame, componentName, version)
	if !ok {
		return usererr.New(usererr.Invalid, fmt.Sprintf("MelonLoader %s is not in Mortar's signed catalog, so Mortar will not install it.", version))
	}
	all, err := l.releases(ctx, c)
	if err != nil {
		return fmt.Errorf("look up MelonLoader releases: %w", err)
	}
	for _, r := range all {
		if v, ok := stable(r); !ok || v != version {
			continue
		}
		if err := l.download(ctx, r, want, dst); err != nil {
			return fmt.Errorf("download MelonLoader %s: %w", version, err)
		}
		return nil
	}
	return fmt.Errorf("GitHub lists no stable MelonLoader release %s", version)
}

func (l Loader) download(ctx context.Context, r github.Release, want, dst string) error {
	var pack github.Asset
	for _, a := range r.Assets {
		if a.Name == asset {
			pack = a
		}
	}
	err := github.Download(ctx, l.HTTP, pack.URL, dst, maxPack, "sha256:"+strings.ToLower(want), nil)
	if errors.Is(err, download.ErrChecksum) {
		return errors.New("the download does not match the checksum in Mortar's signed catalog")
	}
	return err
}
