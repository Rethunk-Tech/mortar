//go:build windows

package runtime

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"golang.org/x/sys/windows"
)

// documentsDir follows a redirected Documents folder (OneDrive) for the signed-in user's own home.
func documentsDir(home string) string {
	if own, err := os.UserHomeDir(); err == nil && fsx.SamePath(own, home) {
		if p, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil && p != "" {
			return p
		}
	}
	return filepath.Join(home, "Documents")
}
