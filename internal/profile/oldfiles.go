package profile

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

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
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	dir, err := s.profileDir(game, id)
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
			if err := os.RemoveAll(filepath.Join(root, k.Name())); err != nil {
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
// file the mod folder has again stays as it is), otherwise they are deleted.
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
				if err := os.Rename(filepath.Join(src, rel), to); err != nil {
					return err
				}
			}
		}
		if err := s.RecordModsSnapshot(game, id); err != nil {
			return err
		}
	}
	return os.RemoveAll(set)
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
