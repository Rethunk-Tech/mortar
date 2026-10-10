package datadir

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// accountHome is the home folder the OS account records, which HOME (USERPROFILE on Windows) can override.
var accountHome = func() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// SharesAccountData reports whether this process would use the account's own Mortar data: HOME is the account's
// home, or the default data folder is the account's default one. A sandbox gives the process a HOME of its own.
func SharesAccountData() bool {
	acct, err := accountHome()
	if err != nil {
		return true
	}
	if home, err := os.UserHomeDir(); err != nil || fsx.SamePath(home, acct) {
		return true
	}
	def, err := defaultDir()
	if err != nil {
		return true
	}
	own := filepath.Join(acct, ".local", "share", "mortar")
	if runtime.GOOS == "windows" {
		own = filepath.Join(acct, "AppData", "Local", "Mortar")
	}
	return filepath.Clean(def) == own
}
