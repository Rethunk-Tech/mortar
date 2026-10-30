package dlwatch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/hostenv"
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
	cmd := exec.CommandContext(ctx, "xdg-user-dir", "DOWNLOAD")
	cmd.Env = hostenv.Environ()
	out, err := cmd.Output()
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
