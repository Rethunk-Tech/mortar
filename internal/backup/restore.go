package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/saves"
)

// Restore copies the named saves from zipPath into l's folder. An empty folders list restores every save in the
// zip. The current saves are zipped first into backupsDir via Saves.
func Restore(zipPath string, l saves.Layout, backupsDir string, folders []string, keep int, now time.Time) error {
	parent := filepath.Dir(l.Dir)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, "restore-*")
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(tmp) }()
	got := saves.Layout{Dir: filepath.Join(tmp, "Saves"), Files: l.Files, Companions: l.Companions}
	want, err := extractSaves(zipPath, got, folders)
	if err != nil {
		return err
	}
	if _, err := Saves(l, backupsDir, keep, now, Cause{Kind: KindRestore}); err != nil {
		return err
	}
	if err := os.MkdirAll(l.Dir, 0o750); err != nil {
		return err
	}
	for _, folder := range want {
		for _, part := range got.Paths(folder) {
			if err := replace(filepath.Join(got.Dir, filepath.FromSlash(part)), filepath.Join(l.Dir, filepath.FromSlash(part))); err != nil {
				return err
			}
		}
	}
	return nil
}

// replace moves src to dst, putting a dst already there back when the move fails.
func replace(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		old := dst + ".mortar-restore"
		if err := fsx.Rename(dst, old); err != nil {
			return err
		}
		if err := fsx.Rename(src, dst); err != nil {
			_ = fsx.Rename(old, dst)
			return err
		}
		return fsx.RemoveAll(old)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return fsx.Rename(src, dst)
}

// extractSaves unpacks zipPath beside got.Dir, its Saves folder, and lists the saves in it that folders names (all
// of them when it names none).
func extractSaves(zipPath string, got saves.Layout, folders []string) ([]string, error) {
	folders = slices.DeleteFunc(slices.Clone(folders), func(f string) bool { return f == "" })
	if err := archive.Extract(zipPath, filepath.Dir(got.Dir)); err != nil {
		return nil, err
	}
	names, err := got.Names()
	if err != nil {
		return nil, err
	}
	for _, f := range folders {
		if f != "" && !slices.Contains(names, f) {
			return nil, fmt.Errorf("backup has no save %q", f)
		}
	}
	if len(folders) == 0 {
		return names, nil
	}
	return slices.DeleteFunc(names, func(n string) bool { return !slices.Contains(folders, n) }), nil
}
