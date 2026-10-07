// Package store keeps each mod archive extracted once under the data folder as a content-addressed blob, names it by
// source keys in an index, and deletes items no profile has used lately.
//
// On disk: blobs/<sha256> holds extracted items, loaders/<game>/<loader>/<version> the loaders' bundled mods, and
// index.json maps game and key to the folder and to the key's source, package and version.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Unused items are deleted this long after their last use.
const retention = 30 * 24 * time.Hour

// CompleteMarker is the empty file an item folder gets once its files are all in place.
const CompleteMarker = ".complete"

const tempPrefix = ".tmp-"

var (
	keyPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
	blobPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const (
	blobsDir   = "blobs"
	loadersDir = "loaders"
)

// Error is a failed store operation. Err stays in the chain, so
// errors.As reaches an *archive.Error and errors.Is reaches a disk-full errno.
type Error struct {
	Game, Key string
	Err       error
}

func (e *Error) Error() string { return fmt.Sprintf("store %s/%s: %v", e.Game, e.Key, e.Err) }

func (e *Error) Unwrap() error { return e.Err }

// DiskFullError is a write that ran out of space; NeedMB is the free space the item needs.
type DiskFullError struct {
	NeedMB int64
	Err    error
}

func (e *DiskFullError) Error() string {
	return fmt.Sprintf("disk full: this item needs about %d MB free: %v", e.NeedMB, e.Err)
}

func (e *DiskFullError) Unwrap() error { return e.Err }

// ErrNotFound reports a key the store does not hold.
var ErrNotFound = errors.New("not in the store")

// ErrIncomplete reports a store folder that never finished extracting.
var ErrIncomplete = errors.New("store item is incomplete")

// Store manages the blob and loader folders under <root> and <root>/index.json.
type Store struct {
	root string
	mu   sync.Mutex
	// idx is the last index read or written, valid while index.json is still the file it came from. Every write
	// replaces the file through a rename, so a new file identity means another writer changed it.
	idxMu   sync.Mutex
	idx     index
	idxFile os.FileInfo
	// UnusedFor is unused-item lifetime; 0 uses the built-in 30 days, negative means keep forever.
	UnusedFor time.Duration
}

// Open returns a store rooted at <datadir>/store.
func Open() (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	return OpenAt(filepath.Join(dir, "store")), nil
}

// OpenAt returns a store rooted at dir.
func OpenAt(dir string) *Store { return &Store{root: dir} }

// LocalKey is the key of a local archive with the given SHA-256.
func LocalKey(sha256Hex string) string { return "local-" + sha256Hex }

// LoaderKey is the key of a loader's entry for one of its versions; its folder is loaders/<game>/<loader>/<version>,
// since games that share a loader id each fetch their own pack (BepInEx 5 for Lethal Company and for Valheim).
func LoaderKey(loaderID, version string) string { return loaderID + "-" + version }

// LoaderOf parses a LoaderKey of a registered loader; ok is false for any other key.
func LoaderOf(key string) (loaderID, version string, ok bool) {
	for _, l := range loader.All() {
		if v, found := strings.CutPrefix(key, l.ID()+"-"); found && v != "" {
			return l.ID(), v, true
		}
	}
	return "", "", false
}

// PackageKey is the key of a package from a source whose names are not valid keys (a Thunderstore Namespace-Name
// has capitals and underscores): a hash of its name and version.
func PackageKey(name, version string) string {
	sum := sha256.Sum256([]byte(name + "@" + version))
	return "pkg-" + hex.EncodeToString(sum[:12])
}

// Meta is what the index records about a key's source, package and version.
func (s *Store) Meta(game, key string) (source, pkg, version string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return "", "", "", false
	}
	r, ok := idx[game][key]
	return r.Source, r.Package, r.Version, ok
}

// Asset is the release asset name recorded for a GitHub item.
func (s *Store) Asset(game, key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return ""
	}
	return idx[game][key].Asset
}

// Keys lists the game's store keys.
func (s *Store) Keys(game string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	keys := slices.Collect(maps.Keys(idx[game]))
	slices.Sort(keys)
	return keys, nil
}

// NexusKey is the key of a file downloaded from Nexus Mods.
func NexusKey(modID, fileID int) string { return fmt.Sprintf("nexus-%d-%d", modID, fileID) }

// NexusFile parses a NexusKey; ok is false for any other key.
func NexusFile(key string) (modID, fileID int, ok bool) {
	n, err := fmt.Sscanf(key, "nexus-%d-%d", &modID, &fileID)
	return modID, fileID, err == nil && n == 2
}

// BridgeKey is the key of the bundled console bridge mod for a version and, when supplied, its asset hash.
func BridgeKey(version string, hashes ...string) string {
	key := "bridge-" + version
	if len(hashes) > 0 && len(hashes[0]) >= 12 {
		key += "-" + strings.ToLower(hashes[0][:12])
	}
	return key
}

func checkGame(id string) error {
	if !game.Valid(id) {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", id))
	}
	return nil
}

func checkKey(id, key string) error {
	if err := checkGame(id); err != nil {
		return err
	}
	if !keyPattern.MatchString(key) || !filepath.IsLocal(key) {
		return fmt.Errorf("invalid store key %q", key)
	}
	return nil
}

// destOf is the folder an item's files live in: a loader's bundle by game, loader and version, any other item by blob.
func (s *Store) destOf(game, key, blob string) (string, error) {
	if id, v, ok := LoaderOf(key); ok {
		if !keyPattern.MatchString(v) || !filepath.IsLocal(id) || !filepath.IsLocal(v) || checkGame(game) != nil {
			return "", fmt.Errorf("invalid store key %q", key)
		}
		return filepath.Join(s.root, loadersDir, game, id, v), nil
	}
	if !blobPattern.MatchString(blob) || !filepath.IsLocal(blob) {
		return "", fmt.Errorf("invalid store blob %q", blob)
	}
	return filepath.Join(s.root, blobsDir, blob), nil
}

// locate is the folder the index gives a key; it is not checked for completeness.
func (s *Store) locate(idx index, game, key string) (string, error) {
	if err := checkKey(game, key); err != nil {
		return "", err
	}
	r, ok := idx[game][key]
	if !ok {
		return "", usererr.Wrap(usererr.NotFound, &Error{Game: game, Key: key, Err: ErrNotFound})
	}
	return s.destOf(game, key, r.Blob)
}

// Path returns the folder to copy into a profile: the item, or the stored content
// root when that relative path still exists inside the item.
func (s *Store) Path(game, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.folder(game, key)
	if err != nil {
		return "", err
	}
	if rel := contentRoot(dir); rel != "" {
		if sub, ok := resolveRoot(dir, rel); ok {
			return sub, nil
		}
	}
	return dir, nil
}

// Dir is the key's whole folder, whatever content root it has.
func (s *Store) Dir(game, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.folder(game, key)
}

func (s *Store) folder(game, key string) (string, error) {
	idx, err := s.viewIndex()
	if err != nil {
		return "", err
	}
	dir, err := s.locate(idx, game, key)
	if err != nil {
		return "", err
	}
	dir = filepath.Clean(dir)
	if !strings.HasPrefix(dir, filepath.Clean(s.root)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid store key %q", key)
	}
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", usererr.Wrap(usererr.NotFound, &Error{Game: game, Key: key, Err: ErrNotFound})
		}
		return "", err
	}
	if !s.completeItem(dir) {
		return "", &Error{Game: game, Key: key, Err: ErrIncomplete}
	}
	return dir, nil
}

// ready reports a key whose folder is complete. An incomplete folder an earlier run left is removed.
func (s *Store) ready(game, key string, idx index) (bool, error) {
	r, ok := idx[game][key]
	if !ok {
		return false, nil
	}
	dir, err := s.destOf(game, key, r.Blob)
	if err != nil {
		return false, err
	}
	return s.dirReady(game, key, dir)
}

func (s *Store) dirReady(game, key, dir string) (bool, error) {
	if s.completeItem(dir) {
		return true, nil
	}
	if exists(dir) {
		if err := fsx.RemoveAll(dir); err != nil {
			return false, &Error{Game: game, Key: key, Err: err}
		}
	}
	return false, nil
}

// AddArchive extracts the archive into the store under its local key, or
// returns the existing key when the same bytes are already there.
func (s *Store) AddArchive(game, archivePath string) (string, error) {
	if err := checkGame(game); err != nil {
		return "", err
	}
	key, err := hashKey(archivePath)
	if err != nil {
		return "", &Error{Game: game, Err: err}
	}
	return key, s.AddArchiveKey(game, key, archivePath)
}

// AddArchiveKey extracts the archive into the store under key, for archives a source names itself. An existing
// key is left as it is, and an archive with the same bytes as another key's shares its blob.
func (s *Store) AddArchiveKey(game, key, archivePath string) error {
	if err := checkKey(game, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admit(game, key, func() (string, error) { return fsx.SHA256(archivePath) }, func(tmp string) error {
		return archive.Extract(archivePath, tmp)
	}, func() int64 {
		n, _ := archive.DeclaredSize(archivePath)
		return n
	})
}

// AddHashedDir copies srcDir into the store under a local key of its contents, or returns the existing
// key when the same tree is already there.
func (s *Store) AddHashedDir(game, srcDir string) (string, error) {
	if err := checkGame(game); err != nil {
		return "", err
	}
	key, err := HashDir(srcDir)
	if err != nil {
		return "", &Error{Game: game, Err: err}
	}
	return key, s.AddDir(game, key, srcDir)
}

// AddDir copies srcDir into the store under key, for entries Mortar builds
// itself. An existing key is left as it is.
func (s *Store) AddDir(game, key, srcDir string) error {
	if err := checkKey(game, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admit(game, key, func() (string, error) {
		if _, _, ok := LoaderOf(key); ok {
			return "", nil
		}
		h, err := HashDir(srcDir)
		return strings.TrimPrefix(h, "local-"), err
	}, func(tmp string) error { return datadir.CopyTree(srcDir, tmp) }, func() int64 {
		n, _ := datadir.Size(srcDir)
		return n
	})
}

// admit makes key name a complete folder. A key already in the index is left as it is; otherwise the content hash
// (empty for a loader bundle, which is named by loader and version) picks the folder, and fill extracts into it
// unless another key already put the same content there.
func (s *Store) admit(game, key string, hash func() (string, error), fill func(tmp string) error, need func() int64) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	if ok, err := s.ready(game, key, idx); err != nil {
		return err
	} else if ok {
		return s.touch(game, key)
	}
	blob, err := hash()
	if err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	dest, err := s.destOf(game, key, blob)
	if err != nil {
		return err
	}
	if ok, err := s.dirReady(game, key, dest); err != nil {
		return err
	} else if !ok {
		if err := s.install(game, key, dest, fill, need); err != nil {
			return err
		}
	}
	return s.bind(game, key, blob)
}

// install fills a temp folder beside the final one and renames it into place,
// removing the temp folder on any failure.
func (s *Store) install(game, key, final string, fill func(tmp string) error, need func() int64) (err error) {
	parent := filepath.Dir(final)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	tmp, err := os.MkdirTemp(parent, tempPrefix)
	if err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	defer func() {
		if err != nil {
			_ = fsx.RemoveAll(tmp)
		}
	}()
	if err = fill(tmp); err == nil {
		stripJunk(tmp)
		marker := filepath.Join(tmp, CompleteMarker)
		if removeErr := os.Remove(marker); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = removeErr
		}
	}
	if err == nil {
		err = syncTree(tmp)
	}
	if err == nil {
		err = writeCompleteMarker(tmp)
	}
	if err == nil {
		err = fsx.Rename(tmp, final)
	}
	if err == nil {
		err = syncPath(parent)
	}
	if err != nil {
		if usererr.IsDiskFull(err) {
			err = usererr.Wrap(usererr.DiskFull, &DiskFullError{NeedMB: need()>>20 + 1, Err: err})
		}
		return &Error{Game: game, Key: key, Err: err}
	}
	s.recordInstalled(game, key, final)
	return nil
}

// completeItem reports whether dir, which must lie under the store's root, holds a whole item.
func (s *Store) completeItem(dir string) bool {
	dir = filepath.Clean(dir)
	if !strings.HasPrefix(dir, filepath.Clean(s.root)+string(os.PathSeparator)) {
		return false
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	marker, err := os.Lstat(filepath.Join(dir, CompleteMarker))
	return err == nil && marker.Mode().IsRegular()
}

func writeCompleteMarker(dir string) error {
	marker := filepath.Join(dir, CompleteMarker)
	if err := datadir.WriteFile(marker, nil, 0o600); err != nil {
		return err
	}
	if err := syncPath(marker); err != nil {
		return err
	}
	return syncPath(dir)
}

func syncTree(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			return syncPath(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.SortFunc(dirs, func(a, b string) int { return len(b) - len(a) })
	for _, dir := range dirs {
		if err := syncPath(dir); err != nil {
			return err
		}
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// HashDir is the content key of a folder: a hash of its file paths and bytes.
func HashDir(root string) (string, error) {
	base, err := fsx.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	type hashed struct{ rel, open string }
	var files []hashed
	err = filepath.WalkDir(root, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if p != root && !datadir.RealDirUnder(base, p) {
				return fs.SkipDir
			}
			return nil
		}
		if datadir.LinkedDir(p, info) {
			return nil
		}
		resolved, err := fsx.EvalSymlinks(p)
		if err != nil {
			return err
		}
		if !datadir.UnderRoot(base, resolved) {
			return fmt.Errorf("%s escapes %s", p, root)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		// The marker is the store's bookkeeping, not content: a folder hashes the same before and after it lands.
		if rel == CompleteMarker {
			return nil
		}
		open := p
		if info.Mode()&os.ModeSymlink != 0 {
			open = resolved
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		files = append(files, hashed{rel: filepath.ToSlash(rel), open: open})
		return nil
	})
	if err != nil {
		return "", err
	}
	slices.SortFunc(files, func(a, b hashed) int { return strings.Compare(a.rel, b.rel) })
	h := sha256.New()
	for _, file := range files {
		if _, err := io.WriteString(h, file.rel); err != nil {
			return "", err
		}
		h.Write([]byte{0})
		f, err := fsx.Open(file.open)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(h, f)
		closeErr := f.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return "", err
		}
		h.Write([]byte{0})
	}
	return LocalKey(hex.EncodeToString(h.Sum(nil))), nil
}

func hashKey(path string) (string, error) {
	sum, err := fsx.SHA256(path)
	if err != nil {
		return "", err
	}
	return LocalKey(sum), nil
}

// record is one key's entry in the index. Blob is the content hash naming the folder under blobs/, empty for a loader
// bundle, whose folder is named by loader and version. Source, Package and Version say where the item came from, in
// the source's own terms, so an item can be found by what it is instead of by the key it was stored under.
type record struct {
	Blob    string `json:"blob,omitempty"`
	Source  string `json:"source,omitempty"`
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	// Asset is the release asset name of a GitHub item, which its key hashes.
	Asset string    `json:"asset,omitempty"`
	Used  time.Time `json:"used"`
	// Size is the folder's size on disk, measured once: an item never changes after it is added.
	Size int64 `json:"size,omitempty"`
}

// index maps game -> key -> record.
type index map[string]map[string]record

// describe is what a key says about its source, package and version.
func describe(key string) (source, pkg, version string) {
	if mod, file, ok := NexusFile(key); ok {
		return "nexus", strconv.Itoa(mod), strconv.Itoa(file)
	}
	if id, v, ok := LoaderOf(key); ok {
		return "loader", id, v
	}
	if rest, ok := strings.CutPrefix(key, "bridge-"); ok {
		v, _, _ := strings.Cut(rest, "-")
		return "mortar", "bridge", v
	}
	if strings.HasPrefix(key, "local-") {
		return "local", "", ""
	}
	if strings.HasPrefix(key, "github-") {
		return "github", "", ""
	}
	return "", "", ""
}

func (s *Store) indexPath() string { return filepath.Join(s.root, "index.json") }

// loadIndex reads the index. One that cannot be parsed is set aside and replaced by an empty one; the blobs it named
// stay on disk until they have been unreferenced for the retention period.
func (s *Store) loadIndex() (index, error) {
	idx, err := s.viewIndex()
	return idx.clone(), err
}

// viewIndex is loadIndex for a caller that only reads: the index it returns is shared and must not be changed.
func (s *Store) viewIndex() (index, error) {
	fi, statErr := os.Stat(s.indexPath())
	s.idxMu.Lock()
	if statErr == nil && s.idxFile != nil && os.SameFile(fi, s.idxFile) && fi.ModTime().Equal(s.idxFile.ModTime()) {
		idx := s.idx
		s.idxMu.Unlock()
		return idx, nil
	}
	s.idxMu.Unlock()
	idx := index{}
	found, err := datadir.ReadJSON(s.indexPath(), &idx)
	if err != nil {
		if !found {
			return nil, err
		}
		log.Printf("store index is corrupt, starting a new one: %v", err)
		if err := fsx.Rename(s.indexPath(), s.indexPath()+".corrupt"); err != nil {
			return nil, err
		}
		return index{}, nil
	}
	if idx == nil {
		idx = index{}
	}
	if statErr == nil {
		s.remember(idx, fi)
	}
	return idx, nil
}

func (s *Store) saveIndex(idx index) error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	// The caller may go on changing idx, so the next read decodes the file again.
	s.remember(nil, nil)
	return datadir.WriteJSON(s.indexPath(), idx)
}

// remember keeps idx, which no caller changes afterwards, as the decoded form of the file fi describes; a nil fi
// forgets it.
func (s *Store) remember(idx index, fi os.FileInfo) {
	s.idxMu.Lock()
	defer s.idxMu.Unlock()
	s.idx, s.idxFile = idx, fi
}

func (idx index) clone() index {
	out := make(index, len(idx))
	for g, recs := range idx {
		out[g] = maps.Clone(recs)
	}
	return out
}

// bind records that key's folder is blob and starts its clock now. What the key says about itself fills the source,
// package and version unless Describe already set them.
func (s *Store) bind(game, key, blob string) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	if idx[game] == nil {
		idx[game] = map[string]record{}
	}
	r := idx[game][key]
	r.Blob, r.Used = blob, time.Now().UTC()
	if dir, err := s.destOf(game, key, blob); err == nil {
		r.Size, _ = datadir.Size(dir)
	}
	if r.Source == "" {
		r.Source, r.Package, r.Version = describe(key)
	}
	idx[game][key] = r
	return s.saveIndex(idx)
}

// Describe records where a stored item came from, for sources whose key does not say (a GitHub release asset).
func (s *Store) Describe(game, key, source, pkg, version, asset string) error {
	if err := checkKey(game, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	r, ok := idx[game][key]
	if !ok {
		return usererr.Wrap(usererr.NotFound, &Error{Game: game, Key: key, Err: ErrNotFound})
	}
	r.Source, r.Package, r.Version, r.Asset = source, pkg, version, asset
	idx[game][key] = r
	return s.saveIndex(idx)
}

// Entry is a key with the folder it names and when it was last used.
type Entry struct {
	Game, Key, Dir string
	LastUsed       time.Time
	// Size is 0 for an item added before sizes were recorded; RecordSizes fills it in.
	Size int64
}

// Entries lists every complete item of every game, for sizes and usage reports.
func (s *Store) Entries() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, game := range slices.Sorted(maps.Keys(idx)) {
		for _, key := range slices.Sorted(maps.Keys(idx[game])) {
			r := idx[game][key]
			if dir, err := s.destOf(game, key, r.Blob); err == nil && s.completeItem(dir) {
				out = append(out, Entry{Game: game, Key: key, Dir: dir, LastUsed: r.Used, Size: r.Size})
			}
		}
	}
	return out, nil
}

// RecordSizes stores measured folder sizes, keyed by game then key, for items that have none.
func (s *Store) RecordSizes(sizes map[string]map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	changed := false
	for game, keys := range sizes {
		for key, n := range keys {
			if r, ok := idx[game][key]; ok && r.Size == 0 && n > 0 {
				r.Size = n
				idx[game][key] = r
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	return s.saveIndex(idx)
}

// Find is the key of the game's item from this source, package and version.
func (s *Store) Find(game, source, pkg, version string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return "", false
	}
	for _, key := range slices.Sorted(maps.Keys(idx[game])) {
		if r := idx[game][key]; r.Source == source && r.Package == pkg && r.Version == version {
			return key, true
		}
	}
	return "", false
}

// touch sets the last use of the keys the index holds to now.
func (s *Store) touch(game string, keys ...string) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, k := range keys {
		if r, ok := idx[game][k]; ok {
			r.Used = now
			idx[game][k] = r
		}
	}
	return s.saveIndex(idx)
}

// Touch sets the last use of the game's keys to now.
func (s *Store) Touch(game string, keys ...string) error {
	for _, k := range keys {
		if err := checkKey(game, k); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.touch(game, keys...)
}

// Collect refreshes the referenced items and deletes every other item whose last use is more than 30 days before
// now, then the blobs nothing names any more. A blob the index never named is kept for the same period, by its
// completion time.
func (s *Store) Collect(referenced map[string][]string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	next := index{}
	dropped := map[string]bool{}
	var errs []error
	for _, game := range slices.Sorted(maps.Keys(idx)) {
		keep := keepSet(referenced[game])
		next[game] = map[string]record{}
		for key, r := range idx[game] {
			// With retention off nothing is deleted and history-only keys are not reported as referenced, so every
			// item counts as used now; turning retention on later starts its clock then.
			if keep[key] || r.Used.IsZero() || s.UnusedFor < 0 {
				r.Used = now
			}
			if !keep[key] && unusedPast(now, r.Used, s.unusedFor()) {
				if err := s.drop(game, key, r); err != nil {
					errs = append(errs, err)
					next[game][key] = r
				} else {
					dropped[r.Blob] = true
				}
				continue
			}
			next[game][key] = r
		}
	}
	errs = append(errs, s.collectBlobs(next, dropped, now))
	errs = append(errs, s.saveIndex(next))
	return errors.Join(errs...)
}

// drop deletes the folder a loader key names; a blob is left for collectBlobs, which knows who else names it.
func (s *Store) drop(game, key string, r record) error {
	s.forget(game, key)
	if _, _, ok := LoaderOf(key); !ok {
		return nil
	}
	dir, err := s.destOf(game, key, r.Blob)
	if err != nil {
		return err
	}
	return fsx.RemoveAll(dir)
}

// collectBlobs deletes the blobs no record of idx names when their last name was just dropped, or when they are older
// than the retention period.
func (s *Store) collectBlobs(idx index, dropped map[string]bool, now time.Time) error {
	named := map[string]bool{}
	for _, keys := range idx {
		for _, r := range keys {
			named[r.Blob] = true
		}
	}
	ents, err := os.ReadDir(filepath.Join(s.root, blobsDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range ents {
		if !e.IsDir() || !blobPattern.MatchString(e.Name()) || named[e.Name()] {
			continue
		}
		if !dropped[e.Name()] {
			info, err := os.Stat(filepath.Join(s.root, blobsDir, e.Name(), CompleteMarker))
			if err != nil || !unusedPast(now, info.ModTime(), s.unusedFor()) {
				continue
			}
		}
		errs = append(errs, fsx.RemoveAll(filepath.Join(s.root, blobsDir, e.Name())))
	}
	return errors.Join(errs...)
}

func (s *Store) unusedFor() time.Duration {
	if s.UnusedFor < 0 {
		return 0
	}
	if s.UnusedFor == 0 {
		return retention
	}
	return s.UnusedFor
}

func unusedPast(now, last time.Time, keep time.Duration) bool {
	if keep <= 0 {
		return false
	}
	return now.Sub(last) > keep
}

func keepSet(keys []string) map[string]bool {
	keep := make(map[string]bool, len(keys))
	for _, k := range keys {
		keep[k] = true
	}
	return keep
}

// Ref is one store item named by game and key.
type Ref struct {
	Game string `json:"game"`
	Key  string `json:"key"`
}

// Unreferenced lists the whole items no referenced key names: the keys Report gives as Unused for the same keep set.
func (s *Store) Unreferenced(referenced map[string][]string) ([]Ref, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	var out []Ref
	for _, game := range slices.Sorted(maps.Keys(idx)) {
		keep := keepSet(referenced[game])
		for _, key := range slices.Sorted(maps.Keys(idx[game])) {
			if _, ok := s.listed(game, key, idx[game][key]); ok && !keep[key] {
				out = append(out, Ref{Game: game, Key: key})
			}
		}
	}
	return out, nil
}

// listed is the item's folder when it holds a whole item, the only kind Report and Unreferenced name; an entry whose
// folder is gone is PruneDangling's, and a partial one is left to the store check.
func (s *Store) listed(game, key string, r record) (string, bool) {
	dir, err := s.destOf(game, key, r.Blob)
	return dir, err == nil && s.completeItem(dir)
}

// PruneDangling drops the index entries no referenced key names whose folder is gone, which nothing could use or
// show; a referenced one stays, since its profile lists it as missing and can fetch it again.
func (s *Store) PruneDangling(referenced map[string][]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	pruned := false
	for game, keys := range idx {
		keep := keepSet(referenced[game])
		for key, r := range keys {
			dir, err := s.destOf(game, key, r.Blob)
			if keep[key] || err != nil {
				continue
			}
			if _, err := os.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
				delete(keys, key)
				pruned = true
			}
		}
	}
	if !pruned {
		return nil
	}
	return s.saveIndex(idx)
}

// Remove deletes the given items and drops them from the index; a blob goes with its last key.
func (s *Store) Remove(refs []Ref) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	dropped := map[string]bool{}
	var errs []error
	for _, it := range refs {
		r, ok := idx[it.Game][it.Key]
		if !ok || checkKey(it.Game, it.Key) != nil {
			continue
		}
		if err := s.drop(it.Game, it.Key, r); err != nil {
			errs = append(errs, err)
			continue
		}
		dropped[r.Blob] = true
		delete(idx[it.Game], it.Key)
	}
	errs = append(errs, s.collectBlobs(idx, dropped, time.Now()))
	errs = append(errs, s.saveIndex(idx))
	return errors.Join(errs...)
}

// Cleanup removes temp folders an interrupted run left behind and returns their paths below the store.
func (s *Store) Cleanup() ([]string, error) {
	parents := []string{filepath.Join(s.root, blobsDir)}
	loaders, err := filepath.Glob(filepath.Join(s.root, loadersDir, "*", "*"))
	if err != nil {
		return nil, err
	}
	for _, l := range loaders {
		if fi, err := os.Stat(l); err == nil && fi.IsDir() {
			parents = append(parents, l)
		}
	}
	var removed []string
	var errs []error
	for _, parent := range parents {
		items, err := os.ReadDir(parent)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, it := range items {
			if !strings.HasPrefix(it.Name(), tempPrefix) {
				continue
			}
			if err := fsx.RemoveAll(filepath.Join(parent, it.Name())); err != nil {
				errs = append(errs, err)
			} else {
				rel, _ := filepath.Rel(s.root, filepath.Join(parent, it.Name()))
				removed = append(removed, filepath.ToSlash(rel))
			}
		}
	}
	return removed, errors.Join(errs...)
}
