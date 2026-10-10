package profile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// heldIndex records when each copy under changed/ was held, by slash path, so a copy whose mod stays out of the
// profile past the trash retention can be deleted with the trash.
const heldIndex = ".mortar-held.json"

func readHeld(dir string) map[string]int64 {
	held := map[string]int64{}
	if b, err := fsx.ReadFile(filepath.Join(dir, heldIndex)); err == nil {
		if json.Unmarshal(b, &held) != nil {
			return map[string]int64{}
		}
	}
	return held
}

func writeHeld(dir string, held map[string]int64) error {
	if len(held) == 0 {
		err := fsx.Remove(filepath.Join(dir, heldIndex))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	b, err := json.Marshal(held)
	if err != nil {
		return err
	}
	return datadir.WriteFile(filepath.Join(dir, heldIndex), b, 0o600)
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

// expireHeld deletes the held copies whose mod has been out of the profile longer than keep. A held copy with no
// record of when it was held (one a restored backup brought) starts its wait now instead of being judged old.
func expireHeld(dir string, keep time.Duration, now time.Time) (int, error) {
	root := filepath.Join(dir, changedDir)
	rec := readHeld(dir)
	onDisk := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		onDisk[filepath.ToSlash(rel)] = true
		return err
	})
	if errors.Is(err, fs.ErrNotExist) {
		return 0, writeHeld(dir, map[string]int64{})
	}
	if err != nil {
		return 0, err
	}
	expired := 0
	for rel := range onDisk {
		at, known := rec[rel]
		if !known {
			rec[rel] = now.Unix()
			continue
		}
		if now.Sub(time.Unix(at, 0)) <= keep {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := fsx.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return expired, err
		}
		removeUp(root, path)
		delete(rec, rel)
		expired++
	}
	for rel := range rec {
		if !onDisk[rel] {
			delete(rec, rel)
		}
	}
	return expired, writeHeld(dir, rec)
}

// expireHeldCopies runs expireHeld for a profile folder and logs what it did; a failure never stops a sync.
func (s *Store) expireHeldCopies(dir string) {
	if _, err := os.Lstat(filepath.Join(dir, changedDir)); err != nil {
		return
	}
	n, err := expireHeld(dir, s.trashKeep(), time.Now())
	if err != nil {
		log.Printf("held copies: %v", err)
		return
	}
	log.Printf("held copies: %d expired", n)
}
