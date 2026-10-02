package savessvc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

// ErrBusy is returned when restore is refused because the game is launching or running.
var ErrBusy = errors.New("stop the game to restore saves")

// ListBackups returns save backups newest first.
func (s *Service) ListBackups() ([]backup.Backup, error) {
	_, dir, err := s.backupDirs()
	if err != nil {
		return nil, err
	}
	return backup.List(dir)
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
	savesDir, backupsDir, err := s.backupDirs()
	if err != nil {
		return err
	}
	keep := backup.DefaultKeep
	if s.settings != nil {
		keep = s.settings.Get().BackupsKept
	}
	return backup.Restore(filepath.Join(backupsDir, name), savesDir, folders, keep, time.Now())
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

// OpenSaveFolder shows one save's folder (a direct child of the Saves folder) in the system file manager.
func (s *Service) OpenSaveFolder(folder string) error {
	if folder == "" || folder != filepath.Base(folder) || folder == "." || folder == ".." {
		return fmt.Errorf("not a save folder: %q", folder)
	}
	savesDir, _, err := s.backupDirs()
	if err != nil {
		return err
	}
	dir := filepath.Join(savesDir, folder)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("save %q not found", folder)
	}
	return datadir.Open(dir)
}

func (s *Service) backupDirs() (savesDir, backupsDir string, err error) {
	savesDir = s.scanner.Dir
	base, err := datadir.Dir()
	if err != nil {
		return "", "", err
	}
	return savesDir, filepath.Join(base, "backups"), nil
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
