package sandbox

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// ExtraFolders validates a path list of folders the player lets a sandboxed game write: each must be absolute, an
// existing folder, and neither the filesystem root nor the home folder itself, which would undo the sandbox.
func ExtraFolders(list, home string) ([]string, error) {
	var out []string
	for _, raw := range filepath.SplitList(list) {
		if raw == "" {
			continue
		}
		p := filepath.Clean(raw)
		switch {
		case !filepath.IsAbs(p):
			return nil, fmt.Errorf("the sandbox folder %q is not an absolute path", raw)
		case p == "/" || p == filepath.Clean(home):
			return nil, fmt.Errorf("the sandbox folder %q would expose everything the sandbox hides", raw)
		case !isDir(p):
			return nil, fmt.Errorf("the sandbox folder %q does not exist", raw)
		case holdsSecrets(p, home):
			return nil, fmt.Errorf("the sandbox folder %q holds or is inside a folder of keys and sign-ins", raw)
		}
		out = append(out, p)
	}
	return out, nil
}

// Allowable is the folder to offer for a path a sandboxed run was refused: the path itself when it is a folder, else
// the nearest existing folder above it. Empty when that is the home folder, a place the sandbox empties as a whole
// (/home, /mnt) or the root, since allowing those would end the sandbox rather than fix one mod, and empty when it
// holds or sits inside a folder of keys and sign-ins: one click must never hand a mod the player's credentials.
func Allowable(path, home string) string {
	p := filepath.Clean(path)
	for !isDir(p) {
		p = filepath.Dir(p)
	}
	if p == filepath.Clean(home) || p == "/" || !strings.HasPrefix(p, "/") {
		return ""
	}
	if slices.Contains(hidden, p) || holdsSecrets(p, home) {
		return ""
	}
	return p
}

// secretDirs are folders under home that hold keys, tokens or browser sign-ins.
var secretDirs = []string{
	".ssh", ".gnupg", ".password-store", ".aws", ".azure", ".kube", ".docker", ".pki", ".mozilla", ".thunderbird",
	".steam/steam/config", ".steam/root/config", ".local/share/Steam/config", ".var/app",
	".config/gh", ".config/gcloud", ".config/rclone", ".config/google-chrome", ".config/chromium",
	".config/BraveSoftware", ".config/microsoft-edge", ".config/vivaldi", ".config/opera", ".config/discord",
	".config/Code", ".local/share/keyrings", ".local/share/kwalletd",
}

// realPath resolves symlinks (/home vs /var/home) so a path and a protected folder compare by where they are.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// holdsSecrets reports whether p is one of secretDirs or Mortar's own data folder, inside one, or above one
// (~/.config holds ~/.config/gh). Both sides are symlink-resolved.
func holdsSecrets(p, home string) bool {
	p, home = realPath(p), realPath(home)
	secrets := make([]string, 0, len(secretDirs)+1)
	for _, d := range secretDirs {
		secrets = append(secrets, realPath(filepath.Join(home, d)))
	}
	if d, err := datadir.Dir(); err == nil {
		secrets = append(secrets, realPath(d))
	}
	for _, s := range secrets {
		if p == s || strings.HasPrefix(p, s+string(filepath.Separator)) || strings.HasPrefix(s, p+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
