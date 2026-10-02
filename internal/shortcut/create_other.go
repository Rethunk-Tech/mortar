//go:build !windows

package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// create writes a desktop entry under the user's applications folder, where launchers and app menus find it.
func create(exe, arg, name string) (string, error) {
	base, err := dataHome()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "applications")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	id := strings.ReplaceAll(strings.TrimPrefix(arg, playFlag), "/", "-")
	path := filepath.Join(dir, "tech.rethunk.Mortar.play-"+id+".desktop")
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Play this Mortar profile
Exec="%s" %s
Icon=mortar
Terminal=false
Categories=Game;
`, desktopValue(name), strings.ReplaceAll(exe, `"`, `\"`), arg)
	if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// desktopValue keeps a name on one line, as the desktop entry format requires.
func desktopValue(s string) string {
	return strings.NewReplacer("\n", " ", "\r", " ").Replace(s)
}

func dataHome() (string, error) {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
