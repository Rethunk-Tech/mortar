package profile

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// heldIndex records when each copy under changed/ was held, by slash path (a copy under replaced/ by its path below
// the profile folder), so a copy whose mod stays out of the
// profile past the trash retention can be deleted with the trash.
const heldIndex = ".mortar-held.json"

func readHeld(dir string) map[string]int64 {
	held := map[string]int64{}
	if _, err := datadir.ReadJSON(filepath.Join(dir, heldIndex), &held); err != nil {
		return map[string]int64{}
	}
	return held
}

func writeHeld(dir string, held map[string]int64) error {
	if len(held) == 0 {
		err := safefs.Remove(filepath.Join(dir, heldIndex))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return datadir.WriteJSON(filepath.Join(dir, heldIndex), held)
}

// markHeld notes that rel was held at now; a copy that comes back is forgotten.
func markHeld(dir, rel string, now time.Time, held bool) error {
	rec := readHeld(dir)
	if held {
		rec[rel] = now.Unix()
	} else {
		delete(rec, rel)
	}
	return writeHeld(dir, rec)
}

// heldOwnersIndex records, by slash path below changed/, the profile path each held copy came from and the entry that
// held it, so a copy comes back only with its own mod. A copy with no record (held by an older Mortar, or brought by a
// backup) sits at its own path and comes back with any mod that lays the path out.
const heldOwnersIndex = ".mortar-held-owners.json"

type heldOwner struct {
	Path string `json:"p"`
	Key  string `json:"k"`
}

func readHeldOwners(dir string) map[string]heldOwner {
	owners := map[string]heldOwner{}
	if _, err := datadir.ReadJSON(filepath.Join(dir, heldOwnersIndex), &owners); err != nil {
		return map[string]heldOwner{}
	}
	return owners
}

func writeHeldOwners(dir string, owners map[string]heldOwner) error {
	if len(owners) == 0 {
		err := safefs.Remove(filepath.Join(dir, heldOwnersIndex))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return datadir.WriteJSON(filepath.Join(dir, heldOwnersIndex), owners)
}

type heldCopy struct{ root, path string }

// expireHeld deletes the held copies whose mod has been out of the profile longer than keep. A held copy with no
// record of when it was held (one a restored backup brought) starts its wait now instead of being judged old.
func expireHeld(dir string, keep time.Duration, now time.Time) (int, error) {
	rec := readHeld(dir)
	// Index key of a copy -> its file.
	onDisk := map[string]heldCopy{}
	for _, root := range []string{changedDir, replacedDir} {
		base := filepath.Join(dir, root)
		err := safefs.WalkRegular(base, func(path string, _ fs.DirEntry) error {
			rel, err := filepath.Rel(base, path)
			key := filepath.ToSlash(rel)
			if root == replacedDir {
				key = replacedDir + "/" + key
			}
			onDisk[key] = heldCopy{base, path}
			return err
		})
		if err != nil {
			return 0, err
		}
	}
	expired := 0
	for key, f := range onDisk {
		at, known := rec[key]
		if !known {
			rec[key] = now.Unix()
			continue
		}
		if now.Sub(time.Unix(at, 0)) <= keep {
			continue
		}
		removeUp(f.root, f.path)
		delete(rec, key)
		expired++
	}
	for key := range rec {
		if _, ok := onDisk[key]; !ok {
			delete(rec, key)
		}
	}
	owners := readHeldOwners(dir)
	for key := range owners {
		if _, ok := rec[key]; !ok {
			delete(owners, key)
		}
	}
	if err := writeHeldOwners(dir, owners); err != nil {
		return expired, err
	}
	return expired, writeHeld(dir, rec)
}

// expireHeldCopies runs expireHeld for a profile folder and logs what it did; a failure never stops a sync.
func (s *Store) expireHeldCopies(dir string) {
	if _, err := os.Lstat(filepath.Join(dir, changedDir)); err != nil {
		if _, err := os.Lstat(filepath.Join(dir, replacedDir)); err != nil {
			return
		}
	}
	n, err := expireHeld(dir, s.trashKeep(), time.Now())
	if err != nil {
		slog.Warn("held copies", "err", err)
		return
	}
	slog.Info("held copies: expiring", "count", n)
}
