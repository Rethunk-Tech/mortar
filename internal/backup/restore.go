package backup

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// ErrZipSlip is returned when a backup zip names a path that would write outside the extract folder.
var ErrZipSlip = errors.New("backup zip contains a path that would escape")

const maxZipFile = 512 << 20

// Restore copies the named save folders from zipPath into savesDir. An empty folders list restores every save
// in the zip. The current Saves folder is zipped first via Saves.
func Restore(zipPath, savesDir string, folders []string, keep int, now time.Time) error {
	if err := checkZip(zipPath); err != nil {
		return err
	}
	backupsDir := filepath.Dir(zipPath)
	if _, err := Saves(savesDir, backupsDir, keep, now, Cause{Kind: KindRestore}); err != nil {
		return err
	}
	parent := filepath.Dir(savesDir)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, "restore-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	want, err := extractSaves(zipPath, tmp, folders)
	if err != nil {
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
			if err := os.Rename(dst, old); err != nil {
				return err
			}
			if err := os.Rename(src, dst); err != nil {
				_ = os.Rename(old, dst)
				return err
			}
			if err := os.RemoveAll(old); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func checkZip(zipPath string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if err := zipNameOK(f.Name); err != nil {
			return err
		}
	}
	return nil
}

func zipNameOK(name string) error {
	rel := filepath.FromSlash(name)
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("%w: %s", ErrZipSlip, name)
	}
	return nil
}

func extractSaves(zipPath, dest string, folders []string) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	allow := map[string]bool{}
	for _, f := range folders {
		if f != "" {
			allow[f] = true
		}
	}
	seen := map[string]bool{}
	var order []string
	for _, f := range zr.File {
		if err := zipNameOK(f.Name); err != nil {
			return nil, err
		}
		folder, _, ok := savePath(f.Name)
		if !ok {
			continue
		}
		if len(allow) > 0 && !allow[folder] {
			continue
		}
		if !seen[folder] {
			seen[folder] = true
			order = append(order, folder)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(filepath.Join(dest, filepath.FromSlash(f.Name)), 0o750); err != nil {
				return nil, err
			}
			continue
		}
		if err := extractFile(f, dest); err != nil {
			return nil, err
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

func extractFile(f *zip.File, dest string) error {
	path := filepath.Join(dest, filepath.FromSlash(f.Name))
	if err := zipNameOK(f.Name); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	out, err := fsx.CreateExcl(path, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(rc, maxZipFile+1))
	if n > maxZipFile {
		return errors.Join(errors.New("file in backup is too large"), out.Close())
	}
	return errors.Join(err, out.Close())
}
