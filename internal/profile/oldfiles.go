package profile

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

// oldFilesDir holds, per entry key and UniqueID, the files an update set aside because the new version no longer
// ships them and oldFilesOnUpdate is ask.
const oldFilesDir = "old-files"

// heldFile is one file set aside by an update: rel is its path in the mod folder, abs where it is now.
type heldFile struct{ uniqueID, rel, abs string }

// OldFile is one file an update set aside, by its mod and its path in that mod's folder.
type OldFile struct {
	UniqueID string `json:"uniqueId"`
	Path     string `json:"path"`
}

// OldFiles is what one entry's last update set aside, waiting for Keep or Delete.
type OldFiles struct {
	Key   string    `json:"key"`
	Label string    `json:"label"`
	Files []OldFile `json:"files"`
}

func (s *Store) oldFilesMode(game string) string {
	if s.OldFilesMode == nil {
		return settings.OldFilesAsk
	}
	return s.OldFilesMode(game)
}

// PendingOldFiles lists the files updates set aside in this profile. Sets left by an entry that has since changed
// version or left the profile are deleted.
func (s *Store) PendingOldFiles(game, id string) ([]OldFiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(dir, oldFilesDir)
	keys, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return []OldFiles{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []OldFiles{}
	for _, k := range keys {
		i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == k.Name() })
		if i < 0 {
			if err := fsx.RemoveAll(filepath.Join(root, k.Name())); err != nil {
				return nil, err
			}
			continue
		}
		set := OldFiles{Key: k.Name(), Label: entryLabel(p.Entries[i]), Files: []OldFile{}}
		mods, err := os.ReadDir(filepath.Join(root, k.Name()))
		if err != nil {
			return nil, err
		}
		for _, m := range mods {
			files, err := relFiles(filepath.Join(root, k.Name(), m.Name()))
			if err != nil {
				return nil, err
			}
			for rel := range files {
				set.Files = append(set.Files, OldFile{UniqueID: m.Name(), Path: filepath.ToSlash(rel)})
			}
		}
		slices.SortFunc(set.Files, func(a, b OldFile) int {
			if a.UniqueID != b.UniqueID {
				return cmp.Compare(a.UniqueID, b.UniqueID)
			}
			return cmp.Compare(a.Path, b.Path)
		})
		if len(set.Files) > 0 {
			out = append(out, set)
		}
	}
	return out, nil
}

// ResolveOldFiles answers the question for entry key: keep puts the set-aside files back in their mod folders (a
// file the mod folder has again stays as it is), otherwise they move to Mortar's trash.
func (s *Store) ResolveOldFiles(game, id, key string, keep bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	if _, err := safeFolder(key); err != nil {
		return err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	set := filepath.Join(dir, oldFilesDir, key)
	if keep {
		mods, err := os.ReadDir(set)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		for _, m := range mods {
			dest, err := s.modFolderLocked(game, id, key, m.Name())
			if err != nil {
				return err
			}
			src := filepath.Join(set, m.Name())
			files, err := relFiles(src)
			if err != nil {
				return err
			}
			for rel := range files {
				to := filepath.Join(dest, rel)
				if exists(to) {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
					return err
				}
				if err := fsx.Rename(filepath.Join(src, rel), to); err != nil {
					return err
				}
			}
		}
		if err := s.RecordModsSnapshot(game, id); err != nil {
			return err
		}
	}
	if keep {
		return fsx.RemoveAll(set)
	}
	_, err = s.trashOldFilesLocked(game, id, key)
	return err
}

// removedOldFilesDir keeps deleted old-file sets, beside removedModsDir.
func (s *Store) removedOldFilesDir(game, id string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(s.removedModsDir(game, id))), "removed-old-files", id)
}

// trashOldFilesLocked moves entry key's set-aside files to Mortar's trash and returns the token
// RestoreOldFiles takes. A set that is already gone yields an empty token.
func (s *Store) trashOldFilesLocked(game, id, key string) (string, error) {
	if err := s.unlocked(game, id); err != nil {
		return "", err
	}
	if _, err := safeFolder(key); err != nil {
		return "", err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return "", err
	}
	src := filepath.Join(dir, oldFilesDir, key)
	if !exists(src) {
		return "", nil
	}
	stamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	dst := filepath.Join(s.removedOldFilesDir(game, id), stamp, key)
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	if err := fsx.Rename(src, dst); err != nil {
		if err := datadir.CopyTree(src, dst); err != nil {
			return "", err
		}
		if err := fsx.RemoveAll(src); err != nil {
			return "", err
		}
	}
	return stamp + "/" + key, nil
}

// TrashOldFiles moves the files an update set aside for entry key to Mortar's trash and returns the token
// RestoreOldFiles takes to bring them back as a pending set.
func (s *Store) TrashOldFiles(game, id, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trashOldFilesLocked(game, id, key)
}

// RestoreOldFiles puts a set TrashOldFiles moved back among the profile's pending old files.
func (s *Store) RestoreOldFiles(game, id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stamp, key, ok := strings.Cut(token, "/")
	if _, err := strconv.ParseInt(stamp, 10, 64); !ok || err != nil {
		return errors.New("invalid restore token")
	}
	if _, err := safeFolder(key); err != nil {
		return errors.New("invalid restore token")
	}
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	src := filepath.Join(s.removedOldFilesDir(game, id), stamp, key)
	if !exists(src) {
		return os.ErrNotExist
	}
	dst := filepath.Join(dir, oldFilesDir, key)
	if exists(dst) {
		return fmt.Errorf("old files for %q are already pending", key)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	if err := fsx.Rename(src, dst); err != nil {
		if err := datadir.CopyTree(src, dst); err != nil {
			return err
		}
		if err := fsx.RemoveAll(src); err != nil {
			return err
		}
	}
	_ = os.Remove(filepath.Dir(src))
	return nil
}

// PendingOldFiles lists files updates set aside because the new version no longer ships them; the UI asks the user
// to keep or delete each set.
func (s *Service) PendingOldFiles(game, id string) ([]OldFiles, error) {
	return s.store.PendingOldFiles(game, id)
}

// ResolveOldFiles keeps (puts back) or deletes the files the last update of entry key set aside.
func (s *Service) ResolveOldFiles(game, id, key string, keep bool) error {
	return s.store.ResolveOldFiles(game, id, key, keep)
}

// TrashOldFiles moves the old files of entry key to Mortar's trash and returns the token RestoreOldFiles takes.
func (s *Service) TrashOldFiles(game, id, key string) (string, error) {
	return s.store.TrashOldFiles(game, id, key)
}

// RestoreOldFiles undoes TrashOldFiles: the set is pending again.
func (s *Service) RestoreOldFiles(game, id, token string) error {
	return s.store.RestoreOldFiles(game, id, token)
}
