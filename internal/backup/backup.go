// Package backup zips a game's Saves folder into the data folder's backups/ before risky operations.
package backup

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// KindLaunch, KindUpdate, KindRestore, and KindManual are Cause.Kind values written beside a zip.
const (
	KindLaunch  = "launch"
	KindUpdate  = "update"
	KindRestore = "restore"
	KindManual  = "manual"
)

// Cause is why a backup was made, stored as a sidecar next to the zip so older timestamp-only names still parse.
type Cause struct {
	Profile string `json:"profile,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Pinned  bool   `json:"pinned,omitempty"`
}

// DefaultKeep is how many backups are retained unless the user chose otherwise.
const DefaultKeep = 5

// MinGap is how recent the newest backup must be to stand in for a new one, so a run of updates cannot
// evict every older backup with copies of the same saves.
const MinGap = 10 * time.Minute

const stamp = "2006-01-02T15-04-05.000"

// Saves zips savesDir into backupsDir/<timestamp>.zip through a temp file and rename, then deletes all but the
// newest keep backups and temp files a crash left. It returns the zip's path (the newest existing one when that is
// under MinGap old and nothing in savesDir changed since), or "" when savesDir does not exist.
func Saves(savesDir, backupsDir string, keep int, now time.Time, cause Cause) (string, error) {
	if _, err := os.Stat(savesDir); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	zips, err := list(backupsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	name := now.UTC().Truncate(time.Millisecond)
	if len(zips) > 0 {
		newest := zips[len(zips)-1]
		if t, err := time.Parse(stamp, strings.TrimSuffix(newest, ".zip")); err == nil {
			// A clock that went back must not name the new zip older than the rest, where prune would take it.
			if !name.After(t) {
				name = t.Add(time.Millisecond)
			}
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
	return finishZip(tmp, backupsDir, savesDir, "", keep, now, name, cause)
}

// Folder zips one save folder into backupsDir the same way Saves does, without the recent-backup stand-in so a
// requested backup is always a new zip.
func Folder(savesDir, backupsDir, folder string, keep int, now time.Time, cause Cause) (string, error) {
	if folder == "" || folder != filepath.Base(folder) || folder == "." || folder == ".." {
		return "", fmt.Errorf("not a save folder: %q", folder)
	}
	dir := filepath.Join(savesDir, folder)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("save %q not found", folder)
	}
	zips, err := list(backupsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	name := now.UTC().Truncate(time.Millisecond)
	if len(zips) > 0 {
		newest := zips[len(zips)-1]
		if t, err := time.Parse(stamp, strings.TrimSuffix(newest, ".zip")); err == nil && !name.After(t) {
			name = t.Add(time.Millisecond)
		}
	}
	if err := os.MkdirAll(backupsDir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(backupsDir, "backup-*.tmp")
	if err != nil {
		return "", err
	}
	return finishZip(tmp, backupsDir, savesDir, folder, keep, now, name, cause)
}

func finishZip(tmp *os.File, backupsDir, savesDir, only string, keep int, now, name time.Time, cause Cause) (string, error) {
	err := writeZip(tmp, savesDir, only)
	if err == nil {
		err = tmp.Sync()
	}
	if err = errors.Join(err, tmp.Close()); err != nil {
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	dst := filepath.Join(backupsDir, name.Format(stamp)+".zip")
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	if err := writeCause(dst, cause); err != nil {
		return dst, errors.Join(err, prune(backupsDir, keep, now))
	}
	return dst, prune(backupsDir, keep, now)
}

func writeCause(zipPath string, cause Cause) error {
	if cause == (Cause{}) {
		return nil
	}
	b, err := json.Marshal(cause)
	if err != nil {
		return err
	}
	return fsx.WriteFile(causePath(zipPath), b, 0o600)
}

func causePath(zipPath string) string {
	return strings.TrimSuffix(zipPath, ".zip") + ".json"
}

func readCause(zipPath string) Cause {
	b, err := fsx.ReadFile(causePath(zipPath))
	if err != nil {
		return Cause{}
	}
	var c Cause
	if json.Unmarshal(b, &c) != nil {
		return Cause{}
	}
	return c
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

func writeZip(w io.Writer, root, only string) error {
	zw := zip.NewWriter(w)
	base := filepath.Base(root)
	walk := root
	if only != "" {
		walk = filepath.Join(root, only)
	}
	err := filepath.WalkDir(walk, func(p string, d fs.DirEntry, err error) error {
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

// SetPinned marks a backup so rotation will retain it.
func SetPinned(backupsDir, name string, pinned bool) error {
	if name == "" || filepath.Base(name) != name || filepath.Ext(name) != ".zip" || strings.Contains(name, "..") {
		return errors.New("invalid backup name")
	}
	path := filepath.Join(backupsDir, name)
	if _, err := os.Stat(path); err != nil {
		return err
	}
	cause := readCause(path)
	cause.Pinned = pinned
	if cause.Profile == "" && cause.Kind == "" && !cause.Pinned {
		if err := os.Remove(causePath(path)); !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeCause(path, cause)
}

// prune removes the oldest unpinned backups beyond keep, and temp files older than MinGap: a backup still being
// written is younger.
func prune(dir string, keep int, now time.Time) error {
	zips, err := list(dir)
	if err != nil {
		return err
	}
	unpinned := 0
	for _, n := range zips {
		if !readCause(filepath.Join(dir, n)).Pinned {
			unpinned++
		}
	}
	var errs []error
	for _, n := range zips {
		if unpinned <= keep {
			break
		}
		if readCause(filepath.Join(dir, n)).Pinned {
			continue
		}
		p := filepath.Join(dir, n)
		errs = append(errs, os.Remove(p))
		if err := os.Remove(causePath(p)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
		unpinned--
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
