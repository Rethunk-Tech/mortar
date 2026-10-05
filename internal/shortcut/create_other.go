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
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/portal"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// launcherID is the desktop file id of a profile's shortcut; it starts with the app id, as the launcher portal requires.
func launcherID(arg string) string {
	return sandbox.AppID + ".play-" + strings.ReplaceAll(strings.TrimPrefix(arg, playFlag), "/", "-") + ".desktop"
}

// create writes a desktop entry under the user's applications folder, where launchers and app menus find it. Inside
// a Flatpak the launcher portal installs it instead, after the user confirms in a system dialog.
func create(exe, arg, name string) (string, error) {
	if sandbox.InFlatpak() {
		id := launcherID(arg)
		return id, portal.InstallLauncher(id, name, "mortar "+arg, nxm.IconPNG())
	}
	execLine := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	base, err := dataHome()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "applications")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(dir, launcherID(arg))
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
	return filepath.Join(base, "applications", launcherID(Arg(game, profile))), nil
}

// Renamed updates the desktop entry for a profile whose name changed. Inside a Flatpak an existing launcher is
// installed again under the new name, which the launcher portal confirms with the user.
func Renamed(game, profile, profileName, gameName string) error {
	if sandbox.InFlatpak() {
		if !validID(game) || !validID(profile) {
			return fmt.Errorf("a shortcut needs a game and a profile")
		}
		arg := Arg(game, profile)
		if ok, err := portal.LauncherExists(launcherID(arg)); err != nil || !ok {
			return err
		}
		return portal.InstallLauncher(launcherID(arg), profileName+" ("+gameName+")", "mortar "+arg, nxm.IconPNG())
	}
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
	if sandbox.InFlatpak() {
		if !validID(game) || !validID(profile) {
			return fmt.Errorf("a shortcut needs a game and a profile")
		}
		return portal.UninstallLauncher(launcherID(Arg(game, profile)))
	}
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
	if sandbox.InFlatpak() {
		if !validID(game) || !validID(profile) {
			return false, fmt.Errorf("a shortcut needs a game and a profile")
		}
		return portal.LauncherExists(launcherID(Arg(game, profile)))
	}
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

// Repoint rewrites each profile shortcut whose program no longer exists to exe, so shortcuts survive Mortar moving
// (an AppImage an integrator renamed, a package replacing a portable copy). A shortcut whose program still exists
// belongs to that copy of Mortar and is left alone.
func Repoint(exe string) error {
	if sandbox.InFlatpak() {
		return nil // The launcher portal writes Flatpak launchers with a flatpak run line, not a path.
	}
	base, err := dataHome()
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(base, "applications", "tech.rethunk.Mortar.play-*.desktop"))
	if err != nil {
		return err
	}
	quoted := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	var errs []error
	for _, p := range paths {
		b, err := fsx.ReadFile(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		lines := strings.SplitAfter(string(b), "\n")
		changed := false
		for i, line := range lines {
			rest, ok := strings.CutPrefix(line, `Exec="`)
			if !ok {
				continue
			}
			end := strings.Index(rest, `" `)
			if end < 0 {
				break
			}
			target := strings.ReplaceAll(rest[:end], `\"`, `"`)
			if _, err := os.Stat(target); err == nil || target == exe {
				break
			}
			lines[i] = "Exec=" + quoted + rest[end+1:]
			changed = true
			break
		}
		if changed {
			errs = append(errs, datadir.WriteFile(p, []byte(strings.Join(lines, "")), 0o600))
		}
	}
	return errors.Join(errs...)
}
