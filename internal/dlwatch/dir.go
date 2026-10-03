package dlwatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// UserDir is the platform Downloads folder: xdg-user-dir DOWNLOAD or ~/Downloads
// on Linux, and the Windows Downloads known folder.
func UserDir() string {
	if runtime.GOOS == "windows" {
		return windowsDownloads()
	}
	return unixDownloads()
}

func unixDownloads() string {
	out, err := exec.Command("xdg-user-dir", "DOWNLOAD").Output()
	if err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" {
			return p
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Downloads")
}
