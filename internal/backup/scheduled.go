package backup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Run counts what one scheduled pass did.
type Run struct {
	Saved     int
	Unchanged int
	Failed    int
}

// Scheduled zips each save folder written since its newest scheduled backup and keeps the newest keep scheduled
// backups per save. A missing Saves folder is an empty run.
func Scheduled(savesDir, backupsDir string, keep int, now time.Time) (Run, error) {
	var run Run
	ents, err := os.ReadDir(savesDir)
	if errors.Is(err, fs.ErrNotExist) {
		return run, nil
	} else if err != nil {
		return run, err
	}
	newest, err := scheduledNewest(backupsDir)
	if err != nil {
		return run, err
	}
	var errs []error
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		folder := e.Name()
		if last, ok := newest[folder]; ok {
			changed, err := lastChange(filepath.Join(savesDir, folder))
			if err != nil {
				run.Failed++
				errs = append(errs, err)
				continue
			}
			if !changed.After(last) {
				run.Unchanged++
				continue
			}
		}
		if _, err := Folder(savesDir, backupsDir, folder, keep, now, Cause{Kind: KindScheduled, Save: folder}); err != nil {
			run.Failed++
			errs = append(errs, err)
			continue
		}
		run.Saved++
	}
	return run, errors.Join(errs...)
}

// LastScheduled is when the newest scheduled backup in backupsDir was taken, zero when there is none.
func LastScheduled(backupsDir string) (time.Time, error) {
	newest, err := scheduledNewest(backupsDir)
	var last time.Time
	for _, t := range newest {
		if t.After(last) {
			last = t
		}
	}
	return last, err
}

// scheduledNewest is each save's newest scheduled backup time.
func scheduledNewest(backupsDir string) (map[string]time.Time, error) {
	zips, err := list(backupsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	newest := map[string]time.Time{}
	for _, n := range zips {
		c := readCause(filepath.Join(backupsDir, n))
		if c.Kind != KindScheduled {
			continue
		}
		if t, err := time.Parse(stamp, strings.TrimSuffix(n, ".zip")); err == nil && t.After(newest[c.Save]) {
			newest[c.Save] = t
		}
	}
	return newest, nil
}
