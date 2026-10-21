package thunderstore

import (
	"bufio"
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

// refreshSide rewrites the side file for a freshly built listing at path, for the packages it already held. It reads the
// listing a package at a time, so a refresh holds no more of it than those packages.
func refreshSide(dir, key, hash, path string) {
	sideMu.Lock()
	old := readSide(dir, key)
	sideMu.Unlock()
	if len(old.Versions) == 0 || old.Hash == hash {
		return
	}
	found := map[string]pkg{}
	err := eachListed(path, func(p *pkg) {
		id := sideID(p.Owner, p.Name)
		if _, asked := old.Versions[id]; !asked {
			return
		}
		if o, seen := found[id]; !seen || !keepsOver(o, *p) {
			found[id] = *p
		}
	})
	if err != nil {
		return
	}
	s := sideFile{Hash: hash, Versions: map[string][]string{}}
	for id, p := range found {
		s.Versions[id] = versionsOf(p)
	}
	sideMu.Lock()
	defer sideMu.Unlock()
	writeSide(dir, key, s)
}

// eachListed hands fn each package of the stored listing at path in turn, keeping none of them.
func eachListed(path string, fn func(*pkg)) error {
	f, err := fsx.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(bufio.NewReader(f))
	if err := expectArray(dec); err != nil {
		return err
	}
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			return err
		}
		fn(&p)
	}
	return nil
}

// findListed is the package namespace-name in the stored listing at path, read a package at a time.
func findListed(path, namespace, name string) (pkg, bool, error) {
	var found pkg
	ok := false
	err := eachListed(path, func(p *pkg) {
		if strings.EqualFold(p.Owner, namespace) && strings.EqualFold(p.Name, name) && (!ok || !keepsOver(found, *p)) {
			found, ok = *p, true
		}
	})
	return found, ok, err
}
