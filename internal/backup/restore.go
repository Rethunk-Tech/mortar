package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Restore copies the named save folders from zipPath into savesDir. An empty folders list restores every save
// in the zip. The current Saves folder is zipped first via Saves.
func Restore(zipPath, savesDir string, folders []string, keep int, now time.Time) error {
	parent := filepath.Dir(savesDir)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, "restore-*")
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(tmp) }()
	want, err := extractSaves(zipPath, tmp, folders)
	if err != nil {
		return err
	}
	backupsDir := filepath.Dir(zipPath)
	if _, err := Saves(savesDir, backupsDir, keep, now, Cause{Kind: KindRestore}); err != nil {
		return err
	}
	if err := os.MkdirAll(savesDir, 0o750); err != nil {
		return err
	}
	for _, folder := range want {
		src := filepath.Join(tmp, "Saves", folder)
		dst := filepath.Join(savesDir, folder)
		if _, err := os.Stat(dst); err == nil {
			old := dst + ".mortar-restore"
			if err := fsx.Rename(dst, old); err != nil {
				return err
			}
			if err := fsx.Rename(src, dst); err != nil {
				_ = fsx.Rename(old, dst)
				return err
			}
			if err := fsx.RemoveAll(old); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := fsx.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func extractSaves(zipPath, dest string, folders []string) ([]string, error) {
	if err := archive.Extract(zipPath, dest); err != nil {
		return nil, err
	}
	allow := map[string]bool{}
	for _, f := range folders {
		if f != "" {
			allow[f] = true
		}
	}
	entries, err := os.ReadDir(filepath.Join(dest, "Saves"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if len(allow) > 0 {
				for folder := range allow {
					return nil, fmt.Errorf("backup has no save %q", folder)
				}
			}
			return nil, nil
		}
		return nil, err
	}
	seen := map[string]bool{}
	var order []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		folder := e.Name()
		if len(allow) > 0 && !allow[folder] {
			continue
		}
		if !seen[folder] {
			seen[folder] = true
			order = append(order, folder)
		}
	}
	if len(allow) > 0 {
		for folder := range allow {
			if !seen[folder] {
				return nil, fmt.Errorf("backup has no save %q", folder)
			}
		}
	}
	slices.Sort(order)
	return order, nil
}
