package thunderstore

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// A community's listing is tens of megabytes, and the loader's own package (BepInExPack and its like) is asked for at
// every start. The version lists of the packages that were asked for by Versions are kept in a small file beside the
// listing, stamped with the index hash it was read from; a refresh rewrites it from the new listing.

type sideFile struct {
	Hash     string              `json:"hash"`
	Versions map[string][]string `json:"versions"`
}

var sideMu sync.Mutex

func sidePath(dir, key string) string { return filepath.Join(dir, key+".versions.json") }

func sideID(namespace, name string) string { return strings.ToLower(namespace + "-" + name) }

func readSide(dir, key string) sideFile {
	var s sideFile
	if b, err := fsx.ReadFile(sidePath(dir, key)); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func writeSide(dir, key string, s sideFile) {
	if b, err := json.Marshal(s); err == nil {
		_ = datadir.WriteFile(sidePath(dir, key), b, 0o644)
	}
}

// sideVersions answers from the side file when the index it was read from is the cached one, however old that is: a
// loader's newest version being one refresh behind is fine, and the refresh a stale index needs runs in the
// background, not on the caller's path.
func (d Driver) sideVersions(ctx context.Context, key, namespace, name, mortarVersion string) ([]string, bool) {
	if !communityKey.MatchString(key) {
		return nil, false
	}
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	var meta cacheMeta
	if b, err := fsx.ReadFile(filepath.Join(dir, key+".meta.json")); err != nil || json.Unmarshal(b, &meta) != nil {
		return nil, false
	}
	if meta.Hash == "" {
		return nil, false
	}
	sideMu.Lock()
	s := readSide(dir, key)
	sideMu.Unlock()
	v, ok := s.Versions[sideID(namespace, name)]
	if !ok || s.Hash != meta.Hash || len(v) == 0 {
		return nil, false
	}
	if d.now().Sub(meta.Fetched) >= refreshAfter {
		d.refreshInBackground(ctx, key, mortarVersion)
	}
	return slices.Clone(v), true
}

var (
	refreshMu    sync.Mutex
	refreshTried = map[string]time.Time{}
)

// refreshInBackground refreshes the community's index once per refreshAfter, which rewrites the side file when the
// listing changes. A refresh that fails (offline) is not retried until the next window.
func (d Driver) refreshInBackground(ctx context.Context, key, mortarVersion string) {
	tried := d.cacheRoot() + "|" + key
	refreshMu.Lock()
	defer refreshMu.Unlock()
	if last, ok := refreshTried[tried]; ok && d.now().Sub(last) < refreshAfter {
		return
	}
	refreshTried[tried] = d.now()
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshTimeout)
	go func() {
		defer cancel()
		_, _ = d.packages(bg, key, source.UserAgentWithSite(mortarVersion))
	}()
}

// refreshTimeout bounds a background index refresh, which downloads every chunk of a large community.
const refreshTimeout = 5 * time.Minute

// rememberVersions adds one package's list, read from the listing the cache holds, to the side file.
func (d Driver) rememberVersions(key, namespace, name string, versions []string) {
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	var meta cacheMeta
	if b, err := fsx.ReadFile(filepath.Join(dir, key+".meta.json")); err != nil || json.Unmarshal(b, &meta) != nil || meta.Hash == "" {
		return
	}
	sideMu.Lock()
	defer sideMu.Unlock()
	s := readSide(dir, key)
	if s.Hash != meta.Hash || s.Versions == nil {
		s = sideFile{Hash: meta.Hash, Versions: map[string][]string{}}
	}
	s.Versions[sideID(namespace, name)] = versions
	writeSide(dir, key, s)
}

// refreshSide rewrites the side file for a freshly built listing, for the packages it already held.
func refreshSide(dir, key, hash string, pk []pkg) {
	sideMu.Lock()
	defer sideMu.Unlock()
	old := readSide(dir, key)
	if len(old.Versions) == 0 {
		return
	}
	s := sideFile{Hash: hash, Versions: map[string][]string{}}
	for _, p := range pk {
		id := sideID(p.Owner, p.Name)
		if _, asked := old.Versions[id]; asked {
			s.Versions[id] = versionsOf(p)
		}
	}
	writeSide(dir, key, s)
}
