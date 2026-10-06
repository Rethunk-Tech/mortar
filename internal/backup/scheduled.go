package backup

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/saves"
)

// Run counts what one scheduled pass did.
type Run struct {
	Saved     int
	Unchanged int
	Failed    int
}

// Scheduled zips each save of l written since its newest scheduled backup and keeps the newest keep scheduled
// backups per save. profile names the profile whose own saves folder l is, "" for the game's shared one, so a save
// folder name in both is tracked and rotated apart. A missing saves folder is an empty run.
func Scheduled(l saves.Layout, backupsDir, profile string, keep int, now time.Time) (Run, error) {
	var run Run
	names, err := l.Names()
	if err != nil || len(names) == 0 {
		return run, err
	}
	newest, err := scheduledNewest(backupsDir, now)
	if err != nil {
		return run, err
	}
	var errs []error
	for _, folder := range names {
		if last, ok := newest[scheduledKey(profile, folder)]; ok {
			changed, err := saveChange(l, folder)
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
		if _, err := Folder(l, backupsDir, folder, keep, now, Cause{Kind: KindScheduled, Save: folder, Profile: profile}); err != nil {
			run.Failed++
			errs = append(errs, err)
			continue
		}
		run.Saved++
	}
	return run, errors.Join(errs...)
}

// LastScheduled is when the newest scheduled backup in backupsDir was taken, zero when there is none. Backups stamped
// in the future are ignored (see scheduledNewest).
func LastScheduled(backupsDir string, now time.Time) (time.Time, error) {
	newest, err := scheduledNewest(backupsDir, now)
	var last time.Time
	for _, t := range newest {
		if t.After(last) {
			last = t
		}
	}
	return last, err
}

// clockSkew is how far ahead of now a backup stamp may be before it is taken as written by a clock that was wrong.
const clockSkew = 5 * time.Minute

// scheduledNewest is each save's newest scheduled backup time. A stamp later than now by more than clockSkew came
// from a clock that ran ahead; counting it would suppress every backup until real time caught up.
func scheduledNewest(backupsDir string, now time.Time) (map[string]time.Time, error) {
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
		key := scheduledKey(c.Profile, c.Save)
		if t, err := time.Parse(stamp, strings.TrimSuffix(n, ".zip")); err == nil && t.After(newest[key]) && !t.After(now.Add(clockSkew)) {
			newest[key] = t
		}
	}
	return newest, nil
}

// scheduledKey is the save a scheduled backup tracks: its folder within the profile's own saves, or the shared ones.
func scheduledKey(profile, save string) string { return profile + "/" + save }
