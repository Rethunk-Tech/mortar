package dlwatch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "xdg-user-dir", "DOWNLOAD").Output()
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
