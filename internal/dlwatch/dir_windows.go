//go:build windows

package dlwatch

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func windowsDownloads() string {
	p, err := windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
	if err == nil && p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Downloads")
}
