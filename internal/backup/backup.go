// Package backup zips a game's Saves folder into the data folder's backups/ before risky operations.
package backup

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// DefaultKeep is how many backups are retained unless the user chose otherwise.
const DefaultKeep = 5

// MinGap is how recent the newest backup must be to stand in for a new one, so a run of updates cannot
// evict every older backup with copies of the same saves.
const MinGap = 10 * time.Minute

const stamp = "2006-01-02T15-04-05.000"

// Saves zips savesDir into backupsDir/<timestamp>.zip through a temp file and rename, then deletes all but the
// newest keep backups and temp files a crash left. It returns the zip's path (the newest existing one when that is
// under MinGap old and nothing in savesDir changed since), or "" when savesDir does not exist.
func Saves(savesDir, backupsDir string, keep int, now time.Time) (string, error) {
	if _, err := os.Stat(savesDir); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	zips, err := list(backupsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if len(zips) > 0 {
		newest := zips[len(zips)-1]
		if t, err := time.Parse(stamp, strings.TrimSuffix(newest, ".zip")); err == nil {
			if age := now.Sub(t); age >= 0 && age < MinGap {
				changed, err := lastChange(savesDir)
				if err != nil {
					return "", err
				}
				if changed.Before(t) {
					return filepath.Join(backupsDir, newest), nil
				}
			}
		}
	}
	if err := os.MkdirAll(backupsDir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(backupsDir, "backup-*.tmp")
	if err != nil {
		return "", err
	}
	err = writeZip(tmp, savesDir)
	if err == nil {
		err = tmp.Sync()
	}
	if err = errors.Join(err, tmp.Close()); err != nil {
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	dst := filepath.Join(backupsDir, now.UTC().Format(stamp)+".zip")
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	return dst, prune(backupsDir, keep, now)
}

// lastChange is the newest modification time in the tree; a folder's covers the files removed from it.
func lastChange(root string) (time.Time, error) {
	var last time.Time
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(last) {
			last = info.ModTime()
		}
		return nil
	})
	return last, err
}

func writeZip(w io.Writer, root string) error {
	zw := zip.NewWriter(w)
	base := filepath.Base(root)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(filepath.Join(base, rel))
		h.Method = zip.Deflate
		zf, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		in, err := fsx.Open(p)
		if err != nil {
			return err
		}
		_, err = io.Copy(zf, in)
		return errors.Join(err, in.Close())
	})
	return errors.Join(err, zw.Close())
}

// list returns the backups' names, oldest first: the timestamp names sort that way.
func list(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var zips []string
	for _, it := range items {
		if strings.HasSuffix(it.Name(), ".zip") {
			zips = append(zips, it.Name())
		}
	}
	slices.Sort(zips)
	return zips, nil
}

// prune removes the oldest backups beyond keep, and temp files older than MinGap: a backup still being written
// is younger.
func prune(dir string, keep int, now time.Time) error {
	zips, err := list(dir)
	if err != nil {
		return err
	}
	var errs []error
	for _, n := range zips[:max(0, len(zips)-keep)] {
		errs = append(errs, os.Remove(filepath.Join(dir, n)))
	}
	tmps, err := filepath.Glob(filepath.Join(dir, "backup-*.tmp"))
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, p := range tmps {
		if info, err := os.Stat(p); err == nil && now.Sub(info.ModTime()) > MinGap {
			errs = append(errs, os.Remove(p))
		}
	}
	return errors.Join(errs...)
}
