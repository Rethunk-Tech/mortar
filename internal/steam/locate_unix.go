//go:build !windows

package steam

import "path/filepath"

func candidates(home string) []string {
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
	}
}
