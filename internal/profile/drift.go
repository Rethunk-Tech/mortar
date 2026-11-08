package profile

import (
	"bytes"
	"cmp"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	ga "github.com/Rethunk-Tech/go-archive"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	modsSnapshotFile = "mods-snapshot.json"
	modsHoldDir      = "mods-hold"
)

// DriftKind is one outside-edit finding in a profile's mods folder.
type DriftKind string

const (
	DriftUnknown DriftKind = "unknown"
	DriftDeleted DriftKind = "deleted"
	DriftChanged DriftKind = "changed"
)

// FolderStat is the cheap fingerprint of one top-level mods folder, ignoring user-written files.
type FolderStat struct {
	Files  int   `json:"files"`
	Size   int64 `json:"size"`
	Newest int64 `json:"newest"`
}

func (st *FolderStat) add(info fs.FileInfo) {
	st.Files++
	st.Size += info.Size()
	st.Newest = max(st.Newest, info.ModTime().UnixNano())
}

// ModsSnapshot is stored beside profile.json after a matching install or an explicit keep.
type ModsSnapshot struct {
	Folders map[string]FolderStat `json:"folders"`
}

// Drift is one unknown, deleted, or changed top-level mods folder.
type Drift struct {
	Kind   DriftKind `json:"kind"`
	Folder string    `json:"folder"`
	Key    string    `json:"key"`
}

// DriftBaseline identifies the recorded mods snapshot that drift is measured against; it changes whenever Mortar
// records a new one, so a caller can tell a remembered drift scan is out of date.
func (s *Store) DriftBaseline(game, id string) string {
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return ""
	}
	info, err := os.Stat(snapshotPath(dir))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + "/" + strconv.FormatInt(info.Size(), 10)
}

func snapshotPath(profileDir string) string {
	return filepath.Join(profileDir, modsSnapshotFile)
}

func holdDir(profileDir string) string {
	return filepath.Join(profileDir, modsHoldDir)
}

func loadSnapshot(profileDir string) (ModsSnapshot, error) {
	var snap ModsSnapshot
	found, err := datadir.ReadJSON(snapshotPath(profileDir), &snap)
	if err != nil {
		return ModsSnapshot{}, err
	}
	if !found {
		return ModsSnapshot{Folders: map[string]FolderStat{}}, nil
	}
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	return snap, nil
}

func writeSnapshot(profileDir string, snap ModsSnapshot) error {
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	return datadir.WriteJSON(snapshotPath(profileDir), snap)
}

func entryFolderName(name string) string {
	return strings.TrimPrefix(name, ".")
}

func isUserWritten(rel string) bool {
	return datadir.WritableRel(rel)
}

// undotDirs drops the leading dot Mortar adds to a switched-off mod's folder, so its files still match the
// store copy, which keeps the plain name.
func undotDirs(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i := range len(parts) - 1 {
		parts[i] = strings.TrimPrefix(parts[i], ".")
	}
	return filepath.FromSlash(strings.Join(parts, "/"))
}

func inPeerSet(inPeer map[string]struct{}, rel string) bool {
	if _, ok := inPeer[rel]; ok {
		return true
	}
	_, ok := inPeer[undotDirs(rel)]
	return ok
}

func walkFolderStat(root, peer string) (FolderStat, error) {
	var inPeer map[string]struct{}
	if peer != "" && peer != root {
		var err error
		if inPeer, err = storeFiles(peer); err != nil {
			return FolderStat{}, err
		}
	}
	prefix := root + string(filepath.Separator)
	var st FolderStat
	var walk func(dir string) error
	walk = func(dir string) error {
		listing, err := listDir(dir)
		if err != nil {
			return err
		}
		for _, name := range listing.Files {
			path := filepath.Join(dir, name)
			rel := strings.TrimPrefix(path, prefix)
			if isUserWritten(rel) {
				continue
			}
			if inPeer != nil && !inPeerSet(inPeer, rel) {
				continue
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			st.add(info)
		}
		for _, name := range listing.Dirs {
			if err := walk(filepath.Join(dir, name)); err != nil {
				return err
			}
		}
		return nil
	}
	return st, walk(root)
}

// dirListing is a directory's entries as last read. Adding, removing or renaming an entry changes the directory's own
// modification time, so a listing taken at the same time is reused; an edit in place changes only the file, so the
// walk still stats every file.
type dirListing struct {
	ModTime int64
	Files   []string
	Dirs    []string
}

// racyListingWindow is how recent a directory's modification time may be before its listing is not kept: an entry
// added within the filesystem's timestamp granularity of the listing would leave the time unchanged.
const racyListingWindow = 2 * time.Second

// gobCache is a map keyed by folder path, read from its gob file on first use and written back atomically when an
// entry changed. A failed write leaves it dirty so the next save retries it, and release holds nothing in memory after
// a save, so the next check reads back from disk instead of keeping tens of megabytes of names between checks.
type gobCache[V any] struct {
	path     func() (string, error)
	gone     func(key string) bool // keys no lookup wanted and whose folder is gone are dropped before a write
	keepUsed bool                  // a key looked up since the process started is kept even if its folder is gone
	release  bool

	mu     sync.Mutex
	byPath map[string]V
	used   map[string]bool
	loaded bool
	dirty  bool
}

// get returns the key's value, from the cache file or an earlier put, and remembers the key was wanted.
func (c *gobCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	if c.keepUsed {
		if c.used == nil {
			c.used = map[string]bool{}
		}
		c.used[key] = true
	}
	v, ok := c.byPath[key]
	return v, ok
}

func (c *gobCache[V]) put(key string, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	if c.byPath == nil {
		c.byPath = map[string]V{}
	}
	c.byPath[key] = v
	c.dirty = true
}

// loadLocked reads the cache file once per process; one that cannot be read or decoded leaves the cache empty, so its
// entries are measured again instead of failing the check.
func (c *gobCache[V]) loadLocked() {
	if c.loaded {
		return
	}
	c.loaded = true
	path, err := c.path()
	if err != nil {
		return
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return
	}
	var disk map[string]V
	if gob.NewDecoder(bytes.NewReader(raw)).Decode(&disk) != nil {
		return
	}
	c.byPath = disk
}

func (c *gobCache[V]) save() {
	c.mu.Lock()
	if !c.dirty {
		c.releaseLocked()
		c.mu.Unlock()
		return
	}
	for key := range c.byPath {
		if c.keepUsed && c.used[key] {
			continue
		}
		if c.gone(key) {
			delete(c.byPath, key)
		}
	}
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(c.byPath)
	c.dirty = false
	c.mu.Unlock()
	if err != nil {
		c.setDirty()
		return
	}
	path, pathErr := c.path()
	if pathErr != nil || os.MkdirAll(filepath.Dir(path), 0o700) != nil || safefs.WriteFileAtomic(path, buf.Bytes(), 0o600) != nil {
		c.setDirty()
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty {
		c.releaseLocked()
	}
}

func (c *gobCache[V]) setDirty() {
	c.mu.Lock()
	c.dirty = true
	c.mu.Unlock()
}

func (c *gobCache[V]) releaseLocked() {
	if c.release {
		c.byPath, c.loaded = nil, false
	}
}

// goneStat and goneLstat are whether a cache key's folder is no longer there; a store item may be a link, so it is
// followed.
func goneStat(key string) bool {
	_, err := os.Stat(key)
	return err != nil
}

func goneLstat(key string) bool {
	_, err := os.Lstat(key)
	return err != nil
}

var dirListings = gobCache[dirListing]{path: dirListingsPath, gone: goneLstat}

func dirListingsPath() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "drift-listings.gob"), nil
}

func listDir(dir string) (dirListing, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return dirListing{}, err
	}
	stamp := info.ModTime().UnixNano()
	if cached, ok := dirListings.get(dir); ok && cached.ModTime == stamp {
		return cached, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dirListing{}, err
	}
	listing := dirListing{ModTime: stamp}
	for _, e := range entries {
		if e.IsDir() {
			listing.Dirs = append(listing.Dirs, e.Name())
		} else {
			listing.Files = append(listing.Files, e.Name())
		}
	}
	if time.Since(info.ModTime()) > racyListingWindow {
		dirListings.put(dir, listing)
	}
	return listing, nil
}

// saveDirListings writes the listings when any changed, dropping those of directories that are gone.
func saveDirListings() {
	dirListings.save()
}

// storeListing is a store item's file list and stats. Store items do not change once installed, so a listing is
// kept on disk across starts and measured again only if the item's folder is replaced.
type storeListing struct {
	ModTime int64
	Files   map[string]struct{}
	Stat    FolderStat
}

var storeListings = gobCache[storeListing]{path: storeListingsPath, gone: goneStat, release: true}

func storeListingsPath() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "store-listings.gob"), nil
}

// saveStoreListings writes the listings measured since the last write, then lets go of them: a drift check reads them
// once, and on a large profile they hold tens of megabytes of file names that the next check reads back from disk.
func saveStoreListings() {
	storeListings.save()
}

func storeListingFor(peer string) (storeListing, error) {
	info, err := os.Stat(peer)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return storeListing{Files: map[string]struct{}{}}, nil
		}
		return storeListing{}, err
	}
	stamp := info.ModTime().UnixNano()
	if cached, ok := storeListings.get(peer); ok && cached.ModTime == stamp {
		return cached, nil
	}
	listing, err := listStoreItem(peer)
	if err != nil {
		return storeListing{}, err
	}
	listing.ModTime = stamp
	storeListings.put(peer, listing)
	return listing, nil
}

func storeFiles(peer string) (map[string]struct{}, error) {
	listing, err := storeListingFor(peer)
	return listing.Files, err
}

// listStoreItem is relFiles and walkFolderStat of a store item in one walk.
func listStoreItem(root string) (storeListing, error) {
	prefix := root + string(filepath.Separator)
	out := storeListing{Files: map[string]struct{}{}}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(path, prefix)
		out.Files[rel] = struct{}{}
		if isUserWritten(rel) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out.Stat.add(info)
		return nil
	})
	return out, err
}

// relFiles lists the files under root by path relative to it, from directory entries alone.
func relFiles(root string) (map[string]struct{}, error) {
	prefix := root + string(filepath.Separator)
	out := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == root {
				return filepath.SkipAll
			}
			return err
		}
		if !d.IsDir() {
			out[strings.TrimPrefix(path, prefix)] = struct{}{}
		}
		return nil
	})
	return out, err
}

func liveFolders(modsDir, heldDir string) (map[string]string, error) {
	names := map[string]string{}
	for _, root := range []string{modsDir, heldDir} {
		ents, found, err := safefs.ReadDirIfExists(root)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, tempPrefix) {
				continue
			}
			key := entryFolderName(name)
			if _, ok := names[key]; !ok {
				names[key] = name
			}
		}
	}
	return names, nil
}

func folderStatAt(modsDir, heldDir, name, peer string) (FolderStat, error) {
	for _, root := range []string{modsDir, heldDir} {
		p := filepath.Join(root, name)
		if datadir.Exists(p) {
			return walkFolderStat(p, peer)
		}
	}
	return FolderStat{}, nil
}

func storePeer(s *Store, game, key string) string {
	if s == nil || s.items == nil || key == "" {
		return ""
	}
	p, err := s.items.Path(game, key)
	if err != nil {
		return ""
	}
	return p
}

func scanDrift(names map[string]string, stats map[string]FolderStat, keys []string, snap ModsSnapshot) []Drift {
	known := map[string]struct{}{}
	for _, k := range keys {
		if k != "" {
			known[k] = struct{}{}
		}
	}
	var out []Drift
	for key, folder := range names {
		if _, ok := known[key]; !ok {
			out = append(out, Drift{Kind: DriftUnknown, Folder: folder, Key: key})
			continue
		}
		prev, ok := snap.Folders[key]
		if !ok {
			continue
		}
		cur := stats[key]
		if cur.Files != prev.Files || cur.Size != prev.Size || cur.Newest != prev.Newest {
			out = append(out, Drift{Kind: DriftChanged, Folder: folder, Key: key})
		}
	}
	for key := range known {
		if _, ok := names[key]; !ok {
			out = append(out, Drift{Kind: DriftDeleted, Folder: key, Key: key})
		}
	}
	slices.SortFunc(out, func(a, b Drift) int {
		if a.Kind != b.Kind {
			return strings.Compare(string(a.Kind), string(b.Kind))
		}
		return strings.Compare(a.Key, b.Key)
	})
	return out
}

func entryKeys(p Profile) []string {
	keys := make([]string, 0, len(p.Entries))
	for _, e := range p.Entries {
		if e.Key != "" && !e.IsOverlay() && e.hasFolder() {
			keys = append(keys, e.Key)
		}
	}
	return keys
}

func safeFolder(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "", errors.New("invalid folder")
	}
	if !ga.WindowsSegment(name) {
		return "", errors.New("invalid folder")
	}
	return name, nil
}

func (s *Store) missingEntryFolders(game, id string) ([]string, error) {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	modsDir := datadir.ModsDirOf(dir)
	var missing []string
	for _, key := range entryKeys(p) {
		if datadir.Exists(filepath.Join(modsDir, key)) || datadir.Exists(filepath.Join(modsDir, "."+key)) {
			continue
		}
		missing = append(missing, key)
	}
	return missing, nil
}

func (s *Store) parkUnknownModsLocked(dir string, p Profile) error {
	known := map[string]struct{}{}
	for _, key := range entryKeys(p) {
		known[key] = struct{}{}
	}
	modsDir := datadir.ModsDirOf(dir)
	ents, found, err := safefs.ReadDirIfExists(modsDir)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	hold := holdDir(dir)
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), tempPrefix) {
			continue
		}
		if _, ok := known[entryFolderName(e.Name())]; ok {
			continue
		}
		if err := os.MkdirAll(hold, 0o700); err != nil {
			return err
		}
		dst := uniqueHoldPath(hold, e.Name())
		if err := safefs.Rename(filepath.Join(modsDir, e.Name()), dst); err != nil {
			return err
		}
	}
	return nil
}

func uniqueHoldPath(dir, name string) string {
	dst := filepath.Join(dir, name)
	if !datadir.Exists(dst) {
		return dst
	}
	for n := 2; ; n++ {
		dst = filepath.Join(dir, fmt.Sprintf("%s-%d", name, n))
		if !datadir.Exists(dst) {
			return dst
		}
	}
}

func (s *Store) unplaceKeys(game, id string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	modsDir := datadir.ModsDirOf(dir)
	for _, key := range keys {
		if err := removeEntryFolders(modsDir, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) liveDriftState(game, id string) (Profile, string, map[string]string, map[string]FolderStat, error) {
	defer saveStoreListings()
	defer saveDirListings()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, "", nil, nil, err
	}
	modsDir := datadir.ModsDirOf(dir)
	held := holdDir(dir)
	names, err := liveFolders(modsDir, held)
	if err != nil {
		return Profile{}, "", nil, nil, err
	}
	stats := make(map[string]FolderStat, len(names))
	keys := make(chan string)
	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for key := range keys {
				st, err := folderStatAt(modsDir, held, names[key], storePeer(s, game, key))
				mu.Lock()
				if err != nil {
					errs = append(errs, err)
				} else {
					stats[key] = st
				}
				mu.Unlock()
			}
		})
	}
	for key := range names {
		keys <- key
	}
	close(keys)
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return Profile{}, "", nil, nil, err
	}
	return p, dir, names, stats, nil
}

// repairSnapshot records the folders of entries profile.json lists that a recorded snapshot lacks: a kill after an
// install wrote profile.json but before its snapshot. Only those entries are added, so drift in any other folder
// stays visible; a profile with no snapshot yet has no baseline to complete.
func (s *Store) repairSnapshot(game, dir string, p Profile) error {
	if _, err := os.Stat(snapshotPath(dir)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	snap, err := loadSnapshot(dir)
	if err != nil {
		return err
	}
	var missing []string
	for _, key := range entryKeys(p) {
		if _, ok := snap.Folders[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	modsDir, held := datadir.ModsDirOf(dir), holdDir(dir)
	names, err := liveFolders(modsDir, held)
	if err != nil {
		return err
	}
	changed := false
	for _, key := range missing {
		folder, ok := names[key]
		if !ok {
			continue
		}
		st, err := folderStatAt(modsDir, held, folder, storePeer(s, game, key))
		if err != nil {
			return err
		}
		snap.Folders[key] = st
		changed = true
	}
	if !changed {
		return nil
	}
	return writeSnapshot(dir, snap)
}

func reconcileSnapshot(snap ModsSnapshot, keys []string, stats, storeStats map[string]FolderStat) ModsSnapshot {
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	if len(snap.Folders) == 0 {
		for _, key := range keys {
			if st, ok := stats[key]; ok {
				snap.Folders[key] = st
			}
		}
		return snap
	}
	for _, key := range keys {
		st, ok := stats[key]
		if !ok {
			continue
		}
		if _, have := snap.Folders[key]; !have {
			snap.Folders[key] = st
			continue
		}
		ss, ok := storeStats[key]
		if ok && st == ss {
			snap.Folders[key] = st
		}
	}
	return snap
}

// RecordModsSnapshot writes the cheap mods-folder fingerprint beside profile.json.
func (s *Store) RecordModsSnapshot(game, id string) error {
	p, dir, _, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return err
	}
	folders := map[string]FolderStat{}
	for _, key := range entryKeys(p) {
		if st, ok := stats[key]; ok {
			folders[key] = st
		}
	}
	return writeSnapshot(dir, ModsSnapshot{Folders: folders})
}

// ScanModsDrift compares the profile's mods folder with the last recorded snapshot.
func (s *Store) ScanModsDrift(game, id string) ([]Drift, error) {
	p, dir, names, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return nil, err
	}
	keys := entryKeys(p)
	snap, err := loadSnapshot(dir)
	if err != nil {
		return nil, err
	}
	storeStats := map[string]FolderStat{}
	for _, key := range keys {
		if peer := storePeer(s, game, key); peer != "" {
			listing, err := storeListingFor(peer)
			if err != nil {
				return nil, err
			}
			storeStats[key] = listing.Stat
		}
	}
	before := maps.Clone(snap.Folders)
	snap = reconcileSnapshot(snap, keys, stats, storeStats)
	if !maps.Equal(before, snap.Folders) {
		if err := writeSnapshot(dir, snap); err != nil {
			return nil, err
		}
	}
	return scanDrift(names, stats, keys, snap), nil
}

func (s *Store) refreshSnapshotKey(game, id, key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	_, dir, _, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return err
	}
	snap, err := loadSnapshot(dir)
	if err != nil {
		return err
	}
	if st, ok := stats[key]; ok {
		snap.Folders[key] = st
	} else {
		delete(snap.Folders, key)
	}
	return writeSnapshot(dir, snap)
}

func (s *Store) restoreDriftEntry(game, id, key string) (Profile, error) {
	return s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		modsDir := datadir.ModsDirOf(dir)
		if err := removeEntryFolders(modsDir, key); err != nil {
			return err
		}
		if err := s.place(game, modsDir, p.Entries[i]); err != nil {
			return err
		}
		return s.relayBase(game, p, dir, key, nil)
	})
}

func (s *Store) revertDriftEntry(game, id, key string) (Profile, error) {
	return s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		e := p.Entries[i]
		if s.items == nil {
			return errors.New("no store")
		}
		src, err := s.items.Path(game, key)
		if err != nil {
			return err
		}
		modsDir := datadir.ModsDirOf(dir)
		scratch, err := os.MkdirTemp(modsDir, tempPrefix)
		if err != nil {
			return err
		}
		if err := datadir.MaterializeTree(src, scratch); err != nil {
			return errors.Join(err, safefs.RemoveAll(scratch))
		}
		none := filepath.Join(scratch, ".no-old")
		for _, m := range e.Mods {
			plain, dotted, err := ModPaths(modsDir, e.Key, m.Folder)
			if err != nil {
				return errors.Join(err, safefs.RemoveAll(scratch))
			}
			cur := plain
			if !datadir.Exists(cur) {
				cur = dotted
			}
			if !datadir.Exists(cur) {
				continue
			}
			if err := carryOver(cur, none, filepath.Join(scratch, filepath.FromSlash(m.Folder))); err != nil {
				return errors.Join(err, safefs.RemoveAll(scratch))
			}
		}
		final, err := materialize(scratch, e)
		if err != nil {
			return errors.Join(err, safefs.RemoveAll(scratch))
		}
		w, err := replaceFolder(modsDir, e.Key, scratch, final)
		if err != nil {
			return errors.Join(err, safefs.RemoveAll(scratch))
		}
		w.commit()
		return s.relayBase(game, p, dir, key, overlaysOf(p.Entries, key))
	})
}

func (s *Store) adoptDriftFolder(ctx context.Context, game, id, folder string) (Profile, error) {
	folder, err := safeFolder(folder)
	if err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	src := filepath.Join(dir, "mods", folder)
	if !datadir.Exists(src) {
		src = filepath.Join(holdDir(dir), folder)
	}
	if !datadir.Exists(src) {
		return Profile{}, os.ErrNotExist
	}
	res, err := s.InstallFolder(ctx, game, id, src)
	if err != nil {
		return Profile{}, err
	}
	_ = safefs.RemoveAll(src)
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return res.Profile, nil
}

// removedModsDir keeps folders taken out of a profile's mods/, apart from trashed profiles, which sit under
// trash/<game>/<profile id>.
func (s *Store) removedModsDir(game, id string) string {
	trash := s.trash
	trash = cmp.Or(trash, filepath.Join(filepath.Dir(s.root), "trash"))
	return filepath.Join(trash, game, "removed-mods", id)
}

// trashModsFolder moves an untracked mods folder aside and returns the token RestoreModsFolder takes.
func (s *Store) trashModsFolder(game, id, folder string) (string, error) {
	folder, err := safeFolder(folder)
	if err != nil {
		return "", err
	}
	if err := s.unlocked(game, id); err != nil {
		return "", err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return "", err
	}
	src := filepath.Join(dir, "mods", folder)
	if !datadir.Exists(src) {
		src = filepath.Join(holdDir(dir), folder)
	}
	if !datadir.Exists(src) {
		return "", os.ErrNotExist
	}
	token := strconv.FormatInt(time.Now().UnixNano(), 10) + "/" + folder
	dst := filepath.Join(s.removedModsDir(game, id), filepath.FromSlash(token))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	if err := safefs.Rename(src, dst); err != nil {
		return "", renameBusy(err)
	}
	return token, s.RecordModsSnapshot(game, id)
}

// restoreSource parses a restore token (a stamp, then the folder or key set aside under it) and returns the profile's
// directory and the set-aside path under trash.
func (s *Store) restoreSource(game, id, token, trash string) (dir, src, name string, err error) {
	stamp, name, ok := strings.Cut(token, "/")
	if _, perr := strconv.ParseInt(stamp, 10, 64); !ok || perr != nil {
		return "", "", "", errors.New("invalid restore token")
	}
	if safe, perr := safeFolder(name); perr != nil || safe != name {
		return "", "", "", errors.New("invalid restore token")
	}
	if err := s.unlocked(game, id); err != nil {
		return "", "", "", err
	}
	dir, err = s.profileDir(game, id)
	if err != nil {
		return "", "", "", err
	}
	src = filepath.Join(trash, stamp, name)
	if !datadir.Exists(src) {
		return "", "", "", os.ErrNotExist
	}
	return dir, src, name, nil
}

// RestoreModsFolder moves a folder trashModsFolder set aside back into mods/.
func (s *Store) RestoreModsFolder(game, id, token string) error {
	dir, src, folder, err := s.restoreSource(game, id, token, s.removedModsDir(game, id))
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, "mods", folder)
	if datadir.Exists(dst) {
		return fmt.Errorf("mods already has a folder named %q", folder)
	}
	if err := safefs.Rename(src, dst); err != nil {
		return renameBusy(err)
	}
	_ = safefs.Remove(filepath.Dir(src))
	return s.RecordModsSnapshot(game, id)
}

// renameBusy words a failed move of a mods folder: the trash lives in the data folder, so the usual cause is a file in
// the folder being open in another program.
func renameBusy(err error) error {
	return usererr.Wrap(usererr.Busy, fmt.Errorf("a file in that folder is in use by another program; close it and try again (%w)", err))
}

//wails:ignore
func (s *Service) ScanModsDrift(game, id string) ([]Drift, error) {
	return s.store.ScanModsDrift(game, id)
}

func (s *Service) KeepDriftChanges(game, id, key string) error {
	return s.store.refreshSnapshotKey(game, id, key)
}

func (s *Service) ForgetDriftEntry(game, id, key string) (Profile, error) {
	p, err := s.store.RemoveEntry(game, id, key)
	if err != nil {
		return Profile{}, err
	}
	_ = s.store.refreshSnapshotKey(game, id, key)
	return p, nil
}

func (s *Service) RestoreDriftEntry(game, id, key string) (Profile, error) {
	return s.store.restoreDriftEntry(game, id, key)
}

func (s *Service) RevertDriftEntry(game, id, key string) (Profile, error) {
	return s.store.revertDriftEntry(game, id, key)
}

func (s *Service) AdoptDriftFolder(ctx context.Context, game, id, folder string) (Profile, error) {
	return s.store.adoptDriftFolder(ctx, game, id, folder)
}

// RemoveDriftFolder sets an untracked folder aside and returns the token RestoreDriftFolder takes to undo it.
func (s *Service) RemoveDriftFolder(game, id, folder string) (string, error) {
	return s.store.trashModsFolder(game, id, folder)
}

// RestoreDriftFolder puts a folder RemoveDriftFolder set aside back into mods/.
func (s *Service) RestoreDriftFolder(game, id, token string) error {
	return s.store.RestoreModsFolder(game, id, token)
}
