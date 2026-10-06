package savessvc

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ErrBusy is returned when a restore or trim is refused because the game is launching or running.
var ErrBusy = usererr.New(usererr.Busy, "stop the game to change save backups")

// ListBackups returns the game's save backups newest first, read from the backups folders on every call. A non-empty
// profile leaves out the backups another profile took; backups that name no profile are listed for every profile.
func (s *Service) ListBackups(game, profile string) ([]backup.Backup, error) {
	id, err := s.saveGame(game)
	if err != nil {
		return nil, err
	}
	reads, err := s.backupReads(id)
	if err != nil {
		return nil, err
	}
	var out []backup.Backup
	seen := map[string]bool{}
	for _, dir := range reads {
		listed, err := backup.List(dir)
		if err != nil {
			return nil, err
		}
		for _, b := range listed {
			if seen[b.Name] || (profile != "" && b.Profile != "" && b.Profile != profile) {
				continue
			}
			seen[b.Name] = true
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b backup.Backup) int {
		return cmp.Compare(b.At, a.At)
	})
	return out, nil
}

// SaveBackups lists, newest first, every backup that contains the save folder.
func (s *Service) SaveBackups(game, save string) ([]backup.Backup, error) {
	all, err := s.ListBackups(game, "")
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(all, func(b backup.Backup) bool {
		return !slices.ContainsFunc(b.Saves, func(sn backup.Snap) bool { return sn.Folder == save })
	}), nil
}

// DeleteBackup removes an unpinned backup, wherever the backups folders hold it.
func (s *Service) DeleteBackup(game, name string) error {
	if err := backupNameOK(name); err != nil {
		return err
	}
	id, err := s.saveGame(game)
	if err != nil {
		return err
	}
	reads, err := s.backupReads(id)
	if err != nil {
		return err
	}
	return backup.Delete(filepath.Dir(findBackup(reads, name)), name)
}

// SetBackupPinned keeps or unkeeps a backup during rotation.
func (s *Service) SetBackupPinned(game, name string, pinned bool) error {
	if err := backupNameOK(name); err != nil {
		return err
	}
	id, err := s.saveGame(game)
	if err != nil {
		return err
	}
	reads, err := s.backupReads(id)
	if err != nil {
		return err
	}
	return backup.SetPinned(filepath.Dir(findBackup(reads, name)), name, pinned)
}

// RestoreBackup copies folders from the named zip into the Saves folder after zipping the current saves.
// An empty folders list restores every save in the zip. The backup taken first rotates against profile's
// saveBackupsKept, as the profile's update and launch backups do.
func (s *Service) RestoreBackup(game, profile, name string, folders []string) error {
	id, err := s.saveGame(game)
	if err != nil {
		return err
	}
	if s.gameBusy(id) {
		return ErrBusy
	}
	if err := backupNameOK(name); err != nil {
		return err
	}
	target, err := s.target(id, profile)
	if err != nil {
		return err
	}
	src := findBackup(target.Reads, name)
	return backup.Restore(src, s.scanners[id].Dir, target.Dir, folders, target.Keep, time.Now())
}

// OpenBackupsFolder shows the backups folder in the system file manager.
func (s *Service) OpenBackupsFolder(game string) error {
	id, err := s.saveGame(game)
	if err != nil {
		return err
	}
	_, dir, err := s.backupDirs(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return datadir.Open(dir)
}

// CreateBackup zips one save and marks the zip kept, with cause kind manual. It reports false, with no backup made,
// when the folder holds no save.
func (s *Service) CreateBackup(game, folder string) (bool, error) {
	id, err := s.saveGame(game)
	if err != nil {
		return false, err
	}
	target, err := s.target(id, "")
	if err != nil {
		return false, err
	}
	now := uniqueBackupTime(target.Reads, time.Now())
	_, err = backup.Folder(s.scanners[id].Dir, target.Dir, folder, target.Keep, now, backup.Cause{Kind: backup.KindManual, Pinned: true})
	if errors.Is(err, backup.ErrNoSaves) {
		return false, nil
	}
	return err == nil, err
}

// uniqueBackupTime moves now forward a millisecond at a time until no backup folder already holds a backup of that
// name: backups are named by their millisecond, and a second one in the same millisecond would replace the first.
func uniqueBackupTime(dirs []string, now time.Time) time.Time {
	for {
		taken := slices.ContainsFunc(dirs, func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, backup.FileName(now)))
			return err == nil
		})
		if !taken {
			return now
		}
		now = now.Add(time.Millisecond)
	}
}

// OpenSaveFolder shows one save's folder (a direct child of the Saves folder) in the system file manager.
func (s *Service) OpenSaveFolder(game, folder string) error {
	id, err := s.saveGame(game)
	if err != nil {
		return err
	}
	savesDir, _, err := s.backupDirs(id)
	if err != nil {
		return err
	}
	dir, err := backup.SaveDir(savesDir, folder)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

// saveGame checks that game has a save folder.
func (s *Service) saveGame(game string) (string, error) {
	if s.scanners[game] != nil {
		return game, nil
	}
	if gamepkg.Find(game) == nil {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", game))
	}
	return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no save folder", game))
}

// target is where the game's backups go, with profileID's overrides when it names one.
func (s *Service) target(gameID, profileID string) (backup.Target, error) {
	set := settings.Defaults()
	if s.settings != nil {
		set = s.settings.Get()
	}
	dir, err := datadir.Dir()
	if err != nil {
		return backup.Target{}, err
	}
	var overrides map[string]string
	if s.profiles != nil && profileID != "" {
		all, err := s.profiles.List(gameID)
		if err != nil {
			return backup.Target{}, err
		}
		if i := slices.IndexFunc(all, func(p profile.Profile) bool { return p.ID == profileID }); i >= 0 {
			overrides = all[i].PrefOverrides()
		}
	}
	return backup.TargetFor(dir, set, gameID, overrides)
}

func (s *Service) backupDirs(gameID string) (savesDir, backupsDir string, err error) {
	target, err := s.target(gameID, "")
	return s.scanners[gameID].Dir, target.Dir, err
}

func (s *Service) backupReads(gameID string) ([]string, error) {
	target, err := s.target(gameID, "")
	return target.Reads, err
}

func findBackup(reads []string, name string) string {
	for _, dir := range reads {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if len(reads) == 0 {
		return name
	}
	return filepath.Join(reads[0], name)
}

func (s *Service) gameBusy(gameID string) bool {
	if s.busy != nil {
		return s.busy()
	}
	if s.Launches == nil {
		return false
	}
	return s.Launches.Busy(gameID)
}

func backupNameOK(name string) error {
	if name == "" || name != filepath.Base(name) || filepath.Ext(name) != ".zip" {
		return fmt.Errorf("invalid backup name")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}
	return nil
}
