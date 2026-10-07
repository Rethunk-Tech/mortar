package profile

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/ids"
)

// A scratch profile is a working copy for an internal check (the crash bisect) that the user never sees. Its folder
// sits beside the profiles under a dot name, which List and every other walk of the profiles folder skip.
const scratchPrefix = ".scratch-"

var scratchPattern = regexp.MustCompile(`^\.scratch-[0-9a-f]{16}$`)

// IsScratch reports whether id names a scratch profile, whose runs are no one's last-played profile.
func IsScratch(id string) bool { return scratchPattern.MatchString(id) }

// ScratchCopy copies the profile, mods/ included, into a scratch profile that no listing shows. DropScratch removes
// it; PurgeScratch removes any a crash left behind.
func (s *Store) ScratchCopy(game, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, srcDir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	gdir, _ := s.gameDir(game)
	newID := scratchPrefix + ids.New()
	dstDir, err := s.profileDir(game, newID)
	if err != nil {
		return Profile{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	dup := src
	dup.ID, dup.Created, dup.Updated = newID, now, now
	if err := copyProfile(gdir, srcDir, dstDir, dup); err != nil {
		return Profile{}, err
	}
	return s.read(game, newID)
}

// DropScratch removes a scratch profile outright; it never goes to the trash.
func (s *Store) DropScratch(game, id string) error {
	if !IsScratch(id) {
		return os.ErrInvalid
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	forgetProfile(filepath.Join(dir, fileName))
	return fsx.RemoveAll(dir)
}

// PurgeScratch removes every game's scratch profiles. It runs at startup, before any check could make one.
func (s *Store) PurgeScratch() error {
	dirs, err := filepath.Glob(filepath.Join(s.root, "*", scratchPrefix+"*"))
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if scratchPattern.MatchString(filepath.Base(d)) {
			forgetProfile(filepath.Join(d, fileName))
			if err := fsx.RemoveAll(d); err != nil {
				return err
			}
		}
	}
	return nil
}
