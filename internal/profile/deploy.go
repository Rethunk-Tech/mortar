package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// placedFile records, in the profile's root, the files SyncPackages put there, so it can take them away again.
const placedFile = ".mortar-packages.json"

// packageFiles are the files the profile's enabled packages lay out below the profile's root, by slash path, each the
// package that wins it (lowest priority first, so a later entry overrides), and how many files each entry wins over an
// earlier one.
func (s *Store) packageFiles(gameID, id string) (files map[string]packageFile, wins map[string]int, err error) {
	info, ok := components.Game(gameID)
	if !ok || info.Deploy != components.DeployProfile {
		return nil, map[string]int{}, nil
	}
	p, _, err := s.readDir(gameID, id)
	if err != nil {
		return nil, nil, err
	}
	ginfo := installerGame(gameID)
	files, wins = map[string]packageFile{}, map[string]int{}
	byFold := map[string]string{}
	for _, e := range p.Entries {
		if e.IsOverlay() || !e.hasPackageEnabled() || e.isTrayEntry() {
			continue
		}
		arch, l, _, err := s.layoutOf(gameID, id, e.StoreKey(), e.Fomod)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", entryLabel(e), err)
		}
		for _, f := range l.Files {
			if (e.File != "" && f.Rel != e.File) || ginfo.IsShared(f.Target) {
				continue
			}
			rel := path.Join(targetPrefix(info, f.Target), path.Clean(f.Rel))
			if !filepath.IsLocal(filepath.FromSlash(rel)) {
				return nil, nil, fmt.Errorf("%s: %s leaves the profile", entryLabel(e), f.Rel)
			}
			// A later entry's file replaces an earlier one that differs only in case: both are one file on Windows.
			fold := fsx.FoldCase(rel)
			if prev, taken := byFold[fold]; taken {
				delete(files, prev)
				wins[e.Key]++
			}
			byFold[fold] = rel
			files[rel] = packageFile{src: filepath.Join(arch.Dir, filepath.FromSlash(f.Src)), key: e.Key}
		}
	}
	return files, wins, nil
}

type packageFile struct{ src, key string }

func targetPrefix(info components.GameInfo, id string) string {
	t, _ := info.Target(id)
	return t.ProfileFolder()
}

// PackageFileOwners maps each file the enabled packages lay out below the profile's root, by slash path, to the key
// of the entry whose copy is there. A game that is redirected has none.
func (s *Store) PackageFileOwners(gameID, id string) (map[string]string, error) {
	files, _, err := s.packageFiles(gameID, id)
	owners := make(map[string]string, len(files))
	for rel, f := range files {
		owners[rel] = f.key
	}
	return owners, err
}

// PackageOverrides counts, for each entry key, the files it wins over an earlier package. A game that is
// redirected holds its mods in the profile's mods folder and has none.
func (s *Store) PackageOverrides(gameID, id string) (map[string]int, error) {
	_, wins, err := s.packageFiles(gameID, id)
	return wins, err
}

// changedDir holds, by slash path, the changed copies of files whose package is no longer laid out (switched off,
// replaced by an update that dropped the file, or removed), so the settings a mod saved come back with the mod.
const changedDir = "changed"

// placedRec is what SyncPackages recorded of a file it placed: the hash of the bytes, and the size and modification
// time the profile's copy had right after, so a later sync tells a copy a mod or the player changed (written back
// from the shared folder, adopted, or edited) from one it laid out.
type placedRec struct {
	Hash  string `json:"h"`
	Size  int64  `json:"s"`
	MTime int64  `json:"t"`
}

func readPlaced(dir string) map[string]placedRec {
	prev := map[string]placedRec{}
	if b, err := fsx.ReadFile(filepath.Join(dir, placedFile)); err == nil {
		if json.Unmarshal(b, &prev) != nil {
			prev = map[string]placedRec{}
		}
	}
	return prev
}

// changedSince reports a placed file whose content is no longer what was recorded.
func changedSince(rec placedRec, dst string) bool {
	st, err := os.Stat(dst)
	if err != nil {
		return false
	}
	if st.Size() == rec.Size && st.ModTime().UnixNano() == rec.MTime {
		return false
	}
	h, err := fsx.SHA256(dst)
	return err != nil || h != rec.Hash
}

// pending is the record written before a file is placed: the hash it will have, and no size, so a crash between the
// copy and the full record still lets a later sync tell the file's own bytes from anyone else's.
func pending(hash string) placedRec { return placedRec{Hash: hash, Size: -1} }

func recordOf(hash, dst string) (placedRec, error) {
	h := hash
	st, err := os.Stat(dst)
	if err != nil {
		return placedRec{}, err
	}
	return placedRec{Hash: h, Size: st.Size(), MTime: st.ModTime().UnixNano()}, nil
}

// SyncPackages lays the enabled packages' files out in the profile, where the loader reads them, and takes away the
// files of packages no longer enabled. A package's config file only seeds the profile: once it is there the player's
// edits stay. So does any other file a mod or the player changed since it was laid out (a script mod's saved
// settings): it is never overwritten from the store, and when its package goes away it is held under `changed/` and
// comes back with the package. A game that is redirected has nothing to sync.
func (s *Store) SyncPackages(gameID, id string) error {
	files, _, err := s.packageFiles(gameID, id)
	if err != nil || files == nil {
		return err
	}
	_, dir, err := s.readDir(gameID, id)
	if err != nil {
		return err
	}
	prev := readPlaced(dir)
	next := maps.Clone(prev)
	var gone []string
	for rel, rec := range prev {
		if _, keep := files[rel]; keep {
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if changedSince(rec, dst) {
			if err := holdChanged(dir, rel); err != nil {
				return err
			}
		}
		removeUp(dir, dst)
		delete(next, rel)
		gone = append(gone, rel)
	}
	if err := holdAdopted(dir, gone, files); err != nil {
		return err
	}
	// The record goes first and names every file this sync may place (pending, with no hash yet), so one that stops part
	// way is still taken away by the next.
	// Every file this sync may place is named first, with the hash it will have, so one that stops part way is still
	// taken away by the next. A profile copy with no record (a restored backup's, or one the player dropped in) that
	// differs from the store is never named: it is kept as it is.
	srcHash := map[string]string{}
	hashOf := func(rel string) (string, error) {
		if h, ok := srcHash[rel]; ok {
			return h, nil
		}
		h, err := fsx.SHA256(files[rel].src)
		srcHash[rel] = h
		return h, err
	}
	for rel := range files {
		if isConfig(rel) {
			continue
		}
		if _, known := next[rel]; known {
			continue
		}
		if info, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			continue
		}
		h, err := hashOf(rel)
		if err != nil {
			return err
		}
		next[rel] = pending(h)
	}
	if err := writePlaced(dir, next); err != nil {
		return err
	}
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if isConfig(rel) {
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		rec, recorded := prev[rel]
		info, statErr := os.Lstat(dst)
		if statErr == nil && info.IsDir() {
			statErr = errors.New("a folder is in the way")
		}
		switch {
		case statErr == nil && !recorded:
			// Not ours unless it is the store's own bytes, which a lost record may have left.
			h, err := hashOf(rel)
			if err != nil {
				return err
			}
			if mine, err := fsx.SHA256(dst); err == nil && mine == h {
				if next[rel], err = recordOf(h, dst); err != nil {
					return err
				}
			}
			continue
		case statErr == nil && changedSince(rec, dst):
			// Still the package's file, with changes that stay; the record keeps it ownable when the package goes.
			continue
		case statErr == nil && placedCurrent(files[rel].src, dst):
			if rec.Size < 0 {
				if next[rel], err = recordOf(rec.Hash, dst); err != nil {
					return err
				}
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if statErr != nil && info == nil {
			if restored, err := restoreChanged(dir, rel); err != nil {
				return err
			} else if restored {
				delete(next, rel)
				continue
			}
		}
		if err := fsx.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := datadir.CopyFile(files[rel].src, dst); err != nil {
			return err
		}
		h, err := hashOf(rel)
		if err != nil {
			return err
		}
		if next[rel], err = recordOf(h, dst); err != nil {
			return err
		}
	}
	if err := seedConfigs(dir, files); err != nil {
		return err
	}
	return writePlaced(dir, next)
}

// ownedFolder is the entry's own folder below the content root of a file path (Mods/mc for Mods/mc/x.cfg), "" for a
// file at the root.
func ownedFolder(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.Join(parts[:2], "/")
}

// holdAdopted holds the files in a folder no package lays out any more that no record names (adopted from play), so
// they leave the profile with the package that owned the folder.
func holdAdopted(dir string, gone []string, files map[string]packageFile) error {
	wanted := map[string]bool{}
	for rel := range files {
		wanted[ownedFolder(rel)] = true
	}
	done := map[string]bool{}
	for _, rel := range gone {
		folder := ownedFolder(rel)
		if folder == "" || wanted[folder] || done[folder] {
			continue
		}
		done[folder] = true
		root := filepath.Join(dir, filepath.FromSlash(folder))
		var held []string
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			r, err := filepath.Rel(dir, path)
			held = append(held, filepath.ToSlash(r))
			return err
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		for _, r := range held {
			if err := holdChanged(dir, r); err != nil {
				return err
			}
		}
		removeUp(dir, root)
	}
	return nil
}

// holdChanged moves a changed copy aside under changed/, replacing an older one.
func holdChanged(dir, rel string) error {
	from := filepath.Join(dir, filepath.FromSlash(rel))
	to := filepath.Join(dir, changedDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	return fsx.Rename(from, to)
}

// restoreChanged puts a held copy back in the profile and reports whether there was one.
func restoreChanged(dir, rel string) (bool, error) {
	from := filepath.Join(dir, changedDir, filepath.FromSlash(rel))
	if _, err := os.Lstat(from); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := fsx.Rename(from, filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
		return false, err
	}
	removeUp(filepath.Join(dir, changedDir), from)
	return true, nil
}

// SeedConfigs copies the config files the enabled packages ship into the profile where it has none yet, so a
// package's settings can be edited before the first launch lays the rest of it out.
func (s *Store) SeedConfigs(gameID, id string) error {
	files, _, err := s.packageFiles(gameID, id)
	if err != nil || files == nil {
		return err
	}
	_, dir, err := s.readDir(gameID, id)
	if err != nil {
		return err
	}
	return seedConfigs(dir, files)
}

func isConfig(rel string) bool { return strings.HasPrefix(rel, "BepInEx/config/") }

func seedConfigs(dir string, files map[string]packageFile) error {
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if !isConfig(rel) {
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Lstat(dst); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if err := datadir.CopyFile(files[rel].src, dst); err != nil {
			return err
		}
	}
	return nil
}

func writePlaced(dir string, rec map[string]placedRec) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return datadir.WriteFile(filepath.Join(dir, placedFile), b, 0o600)
}

// placedCurrent reports a placed file that still matches its source: same size and not older.
func placedCurrent(src, dst string) bool {
	a, err := os.Stat(src)
	if err != nil {
		return false
	}
	b, err := os.Stat(dst)
	return err == nil && a.Size() == b.Size() && !b.ModTime().Before(a.ModTime())
}

// removeUp removes a file, then the folders it leaves empty below root.
func removeUp(root, file string) {
	if fsx.Remove(file) != nil {
		return
	}
	for d := filepath.Dir(file); d != root && datadir.UnderRoot(root, d) && os.Remove(d) == nil; d = filepath.Dir(d) {
	}
}

// hasPackageEnabled reports an entry with no mods to switch off, or with at least one switched on.
func (e Entry) hasPackageEnabled() bool {
	return len(e.Mods) == 0 || slices.ContainsFunc(e.Mods, func(m Component) bool { return e.Enabled(m.ID) })
}
