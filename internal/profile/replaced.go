package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// ReplacedFile is a file SyncPackages moved under replaced/ for a package's own: its path below the profile folder
// and its size.
type ReplacedFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ReplacedFiles lists what SyncPackages replaced in the profile, by path.
func (s *Store) ReplacedFiles(game, id string) ([]ReplacedFile, error) {
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(dir, replacedDir)
	var out []ReplacedFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		out = append(out, ReplacedFile{Path: filepath.ToSlash(rel), Size: info.Size()})
		return err
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	slices.SortFunc(out, func(a, b ReplacedFile) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

// RestoreReplaced puts a replaced file back at its path, over the package's file, and records it as a changed copy so
// SyncPackages keeps it.
func (s *Store) RestoreReplaced(game, id, rel string) error {
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	local := filepath.FromSlash(rel)
	if !filepath.IsLocal(local) || linkOnPath(dir, rel) {
		return fmt.Errorf("%s is not a replaced file", rel)
	}
	from := filepath.Join(dir, replacedDir, local)
	if info, err := os.Lstat(from); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a replaced file", rel)
	}
	to := filepath.Join(dir, local)
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	if err := fsx.Remove(to); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := fsx.Rename(from, to); err != nil {
		return err
	}
	removeUp(filepath.Join(dir, replacedDir), from)
	if err := markHeld(dir, replacedDir+"/"+rel, time.Time{}, false); err != nil {
		return err
	}
	return s.markCopiesChanged(game, id, dir, []string{rel})
}

// DeleteReplaced removes every replaced file of the profile.
func (s *Store) DeleteReplaced(game, id string) error {
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	if err := fsx.RemoveAll(filepath.Join(dir, replacedDir)); err != nil {
		return err
	}
	held := readHeld(dir)
	for key := range held {
		if strings.HasPrefix(key, replacedDir+"/") {
			delete(held, key)
		}
	}
	return writeHeld(dir, held)
}
