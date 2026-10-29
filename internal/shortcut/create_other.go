//go:build !windows

package shortcut

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/userdirs"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
)

// launcherID is the desktop file id of a profile's shortcut; it starts with the app id, as the launcher portal requires.
func launcherID(arg string) string {
	return appversion.AppID + ".play-" + strings.ReplaceAll(strings.TrimPrefix(arg, playFlag), "/", "-") + ".desktop"
}

// create writes a desktop entry under the user's applications folder, where launchers and app menus find it. Inside
// a Flatpak the launcher portal installs it instead, after the user confirms in a system dialog.
func create(exe, arg, name string) (string, error) {
	if flatpak.Inside() {
		id := launcherID(arg)
		return id, installLauncher(id, name, arg)
	}
	execLine := `"` + execArgEscaper.Replace(exe) + `"`
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
	if err := safefs.WriteFileAtomic(path, []byte(entry), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// portalTimeout bounds a portal call, which waits on a dialog the user answers.
const portalTimeout = 5 * time.Minute

// installLauncher asks the launcher portal to add a launcher that plays arg.
func installLauncher(id, name, arg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
	defer cancel()
	return portalErr(portal.InstallLauncher(ctx, portal.Launcher{
		ID:         id,
		Name:       name,
		IconPNG:    nxm.IconPNG(),
		Exec:       []string{"mortar", arg},
		Comment:    "Play this Mortar profile",
		Categories: []string{"Game"},
	}))
}

func launcherExists(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
	defer cancel()
	ok, err := portal.LauncherExists(ctx, id)
	return ok, portalErr(err)
}

// portalErr gives a dismissed portal dialog Mortar's wording.
func portalErr(err error) error {
	if errors.Is(err, portal.ErrCancelled) {
		return errors.New("cancelled in the system dialog")
	}
	return err
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
	if flatpak.Inside() {
		if !validID(game) || !validID(profile) {
			return fmt.Errorf("a shortcut needs a game and a profile")
		}
		arg := Arg(game, profile)
		if ok, err := launcherExists(launcherID(arg)); err != nil || !ok {
			return err
		}
		return installLauncher(launcherID(arg), profileName+" ("+gameName+")", arg)
	}
	path, err := desktopPath(game, profile)
	if err != nil {
		return err
	}
	b, err := safefs.ReadFile(path)
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
	return safefs.WriteFileAtomic(path, []byte(strings.Join(lines, "")), 0o600)
}

// Removed removes the desktop entry for a deleted profile.
func Removed(game, profile string) error {
	if flatpak.Inside() {
		if !validID(game) || !validID(profile) {
			return fmt.Errorf("a shortcut needs a game and a profile")
		}
		ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
		defer cancel()
		return portalErr(portal.UninstallLauncher(ctx, launcherID(Arg(game, profile))))
	}
	path, err := desktopPath(game, profile)
	if err != nil {
		return err
	}
	err = safefs.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func exists(game, profile string) (bool, error) {
	if flatpak.Inside() {
		if !validID(game) || !validID(profile) {
			return false, fmt.Errorf("a shortcut needs a game and a profile")
		}
		return launcherExists(launcherID(Arg(game, profile)))
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

// desktopValue is s as a one-line Desktop Entry string value: a backslash starts an escape there, so it is doubled.
func desktopValue(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", " ", "\r", " ").Replace(s)
}

// execArgEscaper escapes a path for the inside of a quoted Exec argument: the quoted-argument escapes first, then the
// string-value escape that doubles each backslash, and %% so a percent is not read as a field code.
var execArgEscaper = strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`, `%`, `%%`, "\n", `\n`, "\r", `\r`)

// desktopString undoes a key file's string escapes, which wrap the Exec quoting.
var desktopString = strings.NewReplacer(`\\`, `\`, `\s`, " ", `\n`, "\n", `\t`, "\t", `\r`, "\r")

// firstQuoted is the quoted argument an Exec value opens with, its leading quote already cut, and what follows it.
func firstQuoted(s string) (arg, rest string, ok bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				i++
			}
		case '"':
			return strings.ReplaceAll(b.String(), "%%", "%"), s[i+1:], true
		}
		b.WriteByte(s[i])
	}
	return "", "", false
}

func dataHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return userdirs.DataHome(home), nil
}

// RemoveStartMenu deletes every profile's desktop entry, for an uninstall.
func RemoveStartMenu() error {
	base, err := dataHome()
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(base, "applications", appversion.AppID+".play-*.desktop"))
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range paths {
		errs = append(errs, safefs.RemoveAll(p))
	}
	return errors.Join(errs...)
}

// Repoint rewrites each profile shortcut whose program no longer exists to exe, so shortcuts survive Mortar moving
// (an AppImage an integrator renamed, a package replacing a portable copy). A shortcut whose program still exists
// belongs to that copy of Mortar and is left alone.
func Repoint(exe string) error {
	if flatpak.Inside() {
		return nil // The launcher portal writes Flatpak launchers with a flatpak run line, not a path.
	}
	base, err := dataHome()
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(base, "applications", appversion.AppID+".play-*.desktop"))
	if err != nil {
		return err
	}
	quoted := `"` + execArgEscaper.Replace(exe) + `"`
	var errs []error
	for _, p := range paths {
		b, err := safefs.ReadFile(p)
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
			body, nl := strings.CutSuffix(rest, "\n")
			target, tail, ok := firstQuoted(desktopString.Replace(body))
			if !ok {
				break
			}
			if _, err := os.Stat(target); err == nil || target == exe {
				break
			}
			lines[i] = "Exec=" + quoted + desktopValue(tail)
			if nl {
				lines[i] += "\n"
			}
			changed = true
			break
		}
		if changed {
			errs = append(errs, safefs.WriteFileAtomic(p, []byte(strings.Join(lines, "")), 0o600))
		}
	}
	return errors.Join(errs...)
}
