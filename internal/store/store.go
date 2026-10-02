// Package store keeps each mod archive extracted once under the data folder,
// keyed by content or source, and deletes items no profile has used lately.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
)

// Unused items are deleted this long after their last use.
const retention = 30 * 24 * time.Hour

const tempPrefix = ".tmp-"

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

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

// Store manages <root>/<game>/<key>/ folders and <root>/index.json.
type Store struct {
	root string
	mu   sync.Mutex
}

// Open returns a store rooted at <datadir>/store.
func Open() (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	return &Store{root: filepath.Join(dir, "store")}, nil
}

// LocalKey is the key of a local archive with the given SHA-256.
func LocalKey(sha256Hex string) string { return "local-" + sha256Hex }

// SMAPIKey is the key of SMAPI's bundled-mods entry for a SMAPI version.
func SMAPIKey(version string) string { return "smapi-" + version }

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

func (s *Store) gameDir(id string) (string, error) {
	if !game.Valid(id) {
		return "", fmt.Errorf("unknown game %q", id)
	}
	return filepath.Join(s.root, id), nil
}

func (s *Store) itemDir(id, key string) (string, error) {
	dir, err := s.gameDir(id)
	if err != nil {
		return "", err
	}
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid store key %q", key)
	}
	return filepath.Join(dir, key), nil
}

// Path returns the folder to copy into a profile: the item, or the stored content
// root when that relative path still exists inside the item.
func (s *Store) Path(game, key string) (string, error) {
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

func (s *Store) folder(game, key string) (string, error) {
	dir, err := s.itemDir(game, key)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", &Error{Game: game, Key: key, Err: ErrNotFound}
		}
		return "", err
	}
	return dir, nil
}

// AddArchive extracts the archive into the store under its local key, or
// returns the existing key when the same bytes are already there.
func (s *Store) AddArchive(game, archivePath string) (string, error) {
	if _, err := s.gameDir(game); err != nil {
		return "", err
	}
	key, err := hashKey(archivePath)
	if err != nil {
		return "", &Error{Game: game, Err: err}
	}
	return key, s.AddArchiveKey(game, key, archivePath)
}

// AddArchiveKey extracts the archive into the store under key, for archives a source names itself. An existing
// key is left as it is.
func (s *Store) AddArchiveKey(game, key, archivePath string) error {
	if _, err := s.itemDir(game, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if dir, _ := s.itemDir(game, key); exists(dir) {
		return s.touch(game, key)
	}
	return s.install(game, key, func(tmp string) error {
		return archive.Extract(archivePath, tmp, archive.Options{})
	}, func() int64 {
		n, _ := archive.DeclaredSize(archivePath)
		return n
	})
}

// AddHashedDir copies srcDir into the store under a local key of its contents, or returns the existing
// key when the same tree is already there.
func (s *Store) AddHashedDir(game, srcDir string) (string, error) {
	if _, err := s.gameDir(game); err != nil {
		return "", err
	}
	key, err := hashDir(srcDir)
	if err != nil {
		return "", &Error{Game: game, Err: err}
	}
	return key, s.AddDir(game, key, srcDir)
}

// AddDir copies srcDir into the store under key, for entries Mortar builds
// itself. An existing key is left as it is.
func (s *Store) AddDir(game, key, srcDir string) error {
	if _, err := s.itemDir(game, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if dir, _ := s.itemDir(game, key); exists(dir) {
		return s.touch(game, key)
	}
	return s.install(game, key, func(tmp string) error { return datadir.CopyTree(srcDir, tmp) }, func() int64 { return dirSize(srcDir) })
}

// AddDirVerified copies srcDir under key and checks local content keys before installing them.
func (s *Store) AddDirVerified(game, key, srcDir string) error {
	if strings.HasPrefix(key, "local-") {
		got, err := hashDir(srcDir)
		if err != nil {
			return err
		}
		if got != key {
			return fmt.Errorf("store key hash mismatch: got %q, want %q", got, key)
		}
	}
	return s.AddDir(game, key, srcDir)
}

// install fills a temp folder beside the final one and renames it into place,
// removing the temp folder on any failure.
func (s *Store) install(game, key string, fill func(tmp string) error, need func() int64) (err error) {
	gdir, _ := s.gameDir(game)
	final, _ := s.itemDir(game, key)
	if err := os.MkdirAll(gdir, 0o700); err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	tmp, err := os.MkdirTemp(gdir, tempPrefix)
	if err != nil {
		return &Error{Game: game, Key: key, Err: err}
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err = fill(tmp); err == nil {
		stripJunk(tmp)
		err = os.Rename(tmp, final)
	}
	if err != nil {
		if diskFull(err) {
			err = &DiskFullError{NeedMB: need()>>20 + 1, Err: err}
		}
		return &Error{Game: game, Key: key, Err: err}
	}
	return s.touch(game, key)
}

// IsDiskFull reports whether err is a write that ran out of space.
func IsDiskFull(err error) bool { return diskFull(err) }

func diskFull(err error) bool { return errors.Is(err, syscall.ENOSPC) || platformDiskFull(err) }

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hashDir(root string) (string, error) {
	base, err := filepath.EvalSymlinks(root)
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
		resolved, err := filepath.EvalSymlinks(p)
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
		open := p
		if info.Mode()&os.ModeSymlink != 0 {
			st, err := os.Stat(resolved)
			if err != nil {
				return err
			}
			if st.IsDir() {
				return nil
			}
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
	f, err := fsx.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return LocalKey(hex.EncodeToString(h.Sum(nil))), nil
}

func dirSize(dir string) (n int64) {
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, ierr := d.Info(); ierr == nil {
				n += info.Size()
			}
		}
		return nil
	})
	return n
}

// index maps game -> key -> last use.
type index map[string]map[string]time.Time

func (s *Store) indexPath() string { return filepath.Join(s.root, "index.json") }

func (s *Store) loadIndex() (index, error) {
	idx := index{}
	b, err := os.ReadFile(s.indexPath())
	if errors.Is(err, fs.ErrNotExist) {
		return idx, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &idx); err != nil {
		log.Printf("store index is corrupt, rebuilding: %v", err)
		return s.rebuildIndex()
	}
	if idx == nil {
		idx = index{}
	}
	return idx, nil
}

func (s *Store) rebuildIndex() (index, error) {
	now := time.Now().UTC()
	idx := index{}
	games, err := os.ReadDir(s.root)
	if errors.Is(err, fs.ErrNotExist) {
		return idx, s.saveIndex(idx)
	}
	if err != nil {
		return nil, err
	}
	for _, g := range games {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		items, err := os.ReadDir(filepath.Join(s.root, g.Name()))
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !item.IsDir() || strings.HasPrefix(item.Name(), tempPrefix) || !keyPattern.MatchString(item.Name()) {
				continue
			}
			if idx[g.Name()] == nil {
				idx[g.Name()] = map[string]time.Time{}
			}
			idx[g.Name()][item.Name()] = now
		}
	}
	if err := s.saveIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (s *Store) saveIndex(idx index) error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(s.indexPath(), idx)
}

func (s *Store) touch(game string, keys ...string) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	if idx[game] == nil {
		idx[game] = map[string]time.Time{}
	}
	now := time.Now().UTC()
	for _, k := range keys {
		idx[game][k] = now
	}
	return s.saveIndex(idx)
}

// Touch sets the last use of the game's keys to now.
func (s *Store) Touch(game string, keys ...string) error {
	if _, err := s.gameDir(game); err != nil {
		return err
	}
	for _, k := range keys {
		if !keyPattern.MatchString(k) {
			return fmt.Errorf("invalid store key %q", k)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.touch(game, keys...)
}

// Collect refreshes the referenced items and deletes every other item whose
// last use is more than 30 days before now. An item with no recorded use
// starts its clock now.
func (s *Store) Collect(referenced map[string][]string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	games, err := os.ReadDir(s.root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	next := index{}
	var errs []error
	for _, g := range games {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		items, err := os.ReadDir(filepath.Join(s.root, g.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		keep := keepSet(referenced[g.Name()])
		next[g.Name()] = map[string]time.Time{}
		for _, it := range items {
			key := it.Name()
			if !it.IsDir() || strings.HasPrefix(key, tempPrefix) {
				continue
			}
			last, seen := idx[g.Name()][key]
			if keep[key] || !seen {
				last = now
			}
			if !keep[key] && now.Sub(last) > retention {
				if err := os.RemoveAll(filepath.Join(s.root, g.Name(), key)); err != nil {
					errs = append(errs, err)
					next[g.Name()][key] = last
				}
				continue
			}
			next[g.Name()][key] = last
		}
	}
	errs = append(errs, s.saveIndex(next))
	return errors.Join(errs...)
}

func keepSet(keys []string) map[string]bool {
	keep := make(map[string]bool, len(keys))
	for _, k := range keys {
		keep[k] = true
	}
	return keep
}

// Ref is one store folder named by game and key.
type Ref struct {
	Game string `json:"game"`
	Key  string `json:"key"`
}

// Unreferenced lists folders Collect does not treat as in use, using the same keep set.
func (s *Store) Unreferenced(referenced map[string][]string) ([]Ref, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	games, err := os.ReadDir(s.root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var out []Ref
	for _, g := range games {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		items, err := os.ReadDir(filepath.Join(s.root, g.Name()))
		if err != nil {
			return nil, err
		}
		keep := keepSet(referenced[g.Name()])
		for _, it := range items {
			key := it.Name()
			if !it.IsDir() || strings.HasPrefix(key, tempPrefix) {
				continue
			}
			if !keep[key] {
				out = append(out, Ref{Game: g.Name(), Key: key})
			}
		}
	}
	return out, nil
}

// Remove deletes the given store folders and drops them from the index.
func (s *Store) Remove(refs []Ref) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	var errs []error
	for _, it := range refs {
		if !game.Valid(it.Game) || strings.HasPrefix(it.Key, tempPrefix) || !keyPattern.MatchString(it.Key) {
			continue
		}
		errs = append(errs, os.RemoveAll(filepath.Join(s.root, it.Game, it.Key)))
		if idx[it.Game] != nil {
			delete(idx[it.Game], it.Key)
		}
	}
	errs = append(errs, s.saveIndex(idx))
	return errors.Join(errs...)
}

// Cleanup removes temp folders an interrupted run left behind.
func (s *Store) Cleanup() error {
	games, err := os.ReadDir(s.root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, g := range games {
		if !g.IsDir() {
			continue
		}
		items, err := os.ReadDir(filepath.Join(s.root, g.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, it := range items {
			if strings.HasPrefix(it.Name(), tempPrefix) {
				errs = append(errs, os.RemoveAll(filepath.Join(s.root, g.Name(), it.Name())))
			}
		}
	}
	return errors.Join(errs...)
}
