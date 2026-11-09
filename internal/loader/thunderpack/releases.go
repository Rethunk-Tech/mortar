package thunderpack

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/host"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	// Components is the client whose catalog pins the packs; nil means the active one.
	Components *components.Client
	// Keep picks the releases offered; nil keeps every one.
	Keep func(version string) bool
}

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
		if (r.Keep == nil || r.Keep(v)) && r.offered(namespace+"-"+name, v) == nil {
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
	if err := r.offered(namespace+"-"+name, version); err != nil {
		return err
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
	if err := r.checkPin(namespace+"-"+name, version, got); err != nil {
		return err
	}
	return safefs.Rename(got, dst)
}

// checkPin holds the downloaded pack to the SHA-256 the signed catalog pins for its version. Thunderstore publishes no
// digest, so a package the catalog pins nothing of installs unverified, as it did before pins existed. Once the catalog
// pins any version of a package, only pinned versions are offered and installed, and a pinned version whose bytes
// differ is refused.
func (r Releases) checkPin(pack, version, file string) error {
	client := r.client()
	if client == nil {
		return nil
	}
	want, ok := client.Pin(components.PinPack, "", pack, version)
	if !ok {
		slog.Warn("thunderpack: the catalog pins no checksum for this package, so it is installed unverified", "name", r.Name, "version", version)
		return nil
	}
	got, err := safefs.HashFile(file, sha256.New())
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		_ = safefs.Remove(file)
		return usererr.New(usererr.Damaged, fmt.Sprintf("%s %s does not match the checksum in Mortar's signed catalog, so it was not installed.", r.Name, version))
	}
	return nil
}

func (r Releases) client() *components.Client {
	if r.Components != nil {
		return r.Components
	}
	return components.Active()
}

// offered is nil when version of pack may be installed: the catalog pins it, or pins nothing of the package. A version
// released after the catalog was signed waits for the next catalog.
func (r Releases) offered(pack, version string) error {
	client := r.client()
	if client == nil || !client.HasPins(components.PinPack, "", pack) {
		return nil
	}
	if _, ok := client.Pin(components.PinPack, "", pack, version); ok {
		return nil
	}
	return usererr.New(usererr.Outdated, fmt.Sprintf("%s %s is newer than Mortar's signed catalog, which pins the files it installs. It becomes available with the next catalog update (weekly).", r.Name, version))
}
