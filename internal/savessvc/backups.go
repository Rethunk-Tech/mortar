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

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// ErrBusy is returned when restore is refused because the game is launching or running.
var ErrBusy = errors.New("stop the game to restore saves")

// ListBackups returns save backups newest first.
func (s *Service) ListBackups() ([]backup.Backup, error) {
	_, reads, err := s.backupReads()
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
			if seen[b.Name] {
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

// SetBackupPinned keeps or unkeeps a backup during rotation.
func (s *Service) SetBackupPinned(name string, pinned bool) error {
	_, dir, err := s.backupDirs()
	if err != nil {
		return err
	}
	return backup.SetPinned(dir, name, pinned)
}

// RestoreBackup copies folders from the named zip into the Saves folder after zipping the current saves.
// An empty folders list restores every save in the zip.
func (s *Service) RestoreBackup(name string, folders []string) error {
	if s.gameBusy() {
		return ErrBusy
	}
	if err := backupNameOK(name); err != nil {
		return err
	}
	savesDir, reads, err := s.backupReads()
	if err != nil {
		return err
	}
	keep := backup.DefaultKeep
	if s.settings != nil {
		keep = s.settings.Get().BackupsKept
	}
	src := findBackup(reads, name)
	return backup.Restore(src, savesDir, folders, keep, time.Now())
}

// OpenBackupsFolder shows the backups folder in the system file manager.
func (s *Service) OpenBackupsFolder() error {
	_, dir, err := s.backupDirs()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return datadir.Open(dir)
}

// CreateBackup zips one save and marks the zip kept, with cause kind manual.
func (s *Service) CreateBackup(folder string) error {
	savesDir, backupsDir, err := s.backupDirs()
	if err != nil {
		return err
	}
	keep := backup.DefaultKeep
	if s.settings != nil {
		keep = s.settings.Get().BackupsKept
	}
	_, reads, err := s.backupReads()
	if err != nil {
		return err
	}
	now := uniqueBackupTime(reads, time.Now())
	_, err = backup.Folder(savesDir, backupsDir, folder, keep, now, backup.Cause{Kind: backup.KindManual, Pinned: true})
	return err
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
func (s *Service) OpenSaveFolder(folder string) error {
	savesDir, _, err := s.backupDirs()
	if err != nil {
		return err
	}
	dir, err := backup.SaveDir(savesDir, folder)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}

func (s *Service) backupDirs() (savesDir, backupsDir string, err error) {
	savesDir = s.scanner.Dir
	loc := ""
	if s.settings != nil {
		loc = settings.Resolve(s.settings.Get(), "backupLocation", settings.GameStardew, nil)
	}
	backupsDir, _, err = backup.Locations(loc)
	return savesDir, backupsDir, err
}

func (s *Service) backupReads() (savesDir string, reads []string, err error) {
	savesDir = s.scanner.Dir
	loc := ""
	if s.settings != nil {
		loc = settings.Resolve(s.settings.Get(), "backupLocation", settings.GameStardew, nil)
	}
	_, reads, err = backup.Locations(loc)
	return savesDir, reads, err
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

func (s *Service) gameBusy() bool {
	if s.busy != nil {
		return s.busy()
	}
	if s.Launches == nil {
		return false
	}
	st, err := s.Launches.Status("stardew")
	if err != nil {
		return false
	}
	return st.State == launchsvc.Launching || st.State == launchsvc.Running
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
