//go:build !windows

package steam

import "path/filepath"

func candidates(home string) []string {
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
		filepath.Join(home, ".steam", "debian-installation"),
		filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"),
	}
}
