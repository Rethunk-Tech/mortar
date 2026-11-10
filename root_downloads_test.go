//go:build !windows

package main

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// Mortar's own download folder is the settings' archive folder when it is absolute, else one below the data folder;
// the folders offered are each listed once, so a folder named both in the settings and by the desktop is not watched
// or offered twice.
func TestDownloadDirsFollowSettingsAndStayUnique(t *testing.T) {
	prev := userDownloads
	downloads := t.TempDir()
	userDownloads = func() string { return downloads }
	t.Cleanup(func() { userDownloads = prev })
	store, _ := testenv.Env(t)
	dataDir := t.TempDir()
	want := filepath.Join(dataDir, "downloads")
	if got := archiveDir(store, dataDir); got != want {
		t.Errorf("archiveDir = %q, want %q", got, want)
	}
	abs := t.TempDir()
	if _, err := store.Update(func(s *settings.Settings) {
		s.DownloadFolder = abs
		s.WatchFolders = abs + string(filepath.ListSeparator) + downloads
	}); err != nil {
		t.Fatal(err)
	}
	if got := archiveDir(store, dataDir); got != abs {
		t.Errorf("an absolute archive folder = %q, want %q", got, abs)
	}
	dirs := downloadDirs(store, dataDir)
	if !slices.Equal(dirs, []string{abs, downloads}) {
		t.Fatalf("downloadDirs = %v, want the archive folder and the user's Downloads folder once each", dirs)
	}
}
