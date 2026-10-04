//go:build !windows

package shortcut

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// create writes a desktop entry under the user's applications folder, where launchers and app menus find it.
func create(exe, arg, name string) (string, error) {
	execLine := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	if sandbox.InFlatpak() {
		execLine = sandbox.HostExec
	}
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
Exec=%s %s
Icon=mortar
Terminal=false
Categories=Game;
`, desktopValue(name), execLine, arg)
	if err := datadir.WriteFile(path, []byte(entry), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func desktopPath(game, profile string) (string, error) {
	if !validID(game) || !validID(profile) {
		return "", fmt.Errorf("a shortcut needs a game and a profile")
	}
	base, err := dataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "applications", "tech.rethunk.Mortar.play-"+game+"-"+profile+".desktop"), nil
}

// Renamed updates the desktop entry for a profile whose name changed.
func Renamed(game, profile, profileName, gameName string) error {
	path, err := desktopPath(game, profile)
	if err != nil {
		return err
	}
	b, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.SplitAfter(string(b), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "Name=") {
			suffix := ""
			if strings.HasSuffix(line, "\n") {
				suffix = "\n"
			}
			lines[i] = "Name=" + desktopValue(profileName+" ("+gameName+")") + suffix
			replaced = true
			break
		}
	}
	if !replaced {
		return fmt.Errorf("desktop entry %s has no Name", path)
	}
	return datadir.WriteFile(path, []byte(strings.Join(lines, "")), 0o600)
}

// Removed removes the desktop entry for a deleted profile.
func Removed(game, profile string) error {
	path, err := desktopPath(game, profile)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func exists(game, profile string) (bool, error) {
	path, err := desktopPath(game, profile)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil && info.Mode().IsRegular(), err
}

// desktopValue keeps a name on one line, as the desktop entry format requires.
func desktopValue(s string) string {
	return strings.NewReplacer("\n", " ", "\r", " ").Replace(s)
}

func dataHome() (string, error) {
	if sandbox.InFlatpak() {
		// The host's menus read ~/.local/share/applications; XDG_DATA_HOME is Mortar's private sandbox folder.
		return sandbox.HostDataHome()
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// RemoveStartMenu deletes every profile's desktop entry, for an uninstall.
func RemoveStartMenu() error {
	base, err := dataHome()
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(base, "applications", "tech.rethunk.Mortar.play-*.desktop"))
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range paths {
		errs = append(errs, fsx.RemoveAll(p))
	}
	return errors.Join(errs...)
}
