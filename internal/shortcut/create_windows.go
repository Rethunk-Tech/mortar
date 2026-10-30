//go:build windows

package shortcut

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/lnk"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// create writes a .lnk in the Start menu's Mortar folder through the shell's own shortcut object.
func create(exe, arg, name string) (string, error) {
	dir, err := startMenuDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	path, err := freeLinkPath(dir, fileName(name), arg)
	if err != nil {
		return "", err
	}
	if err := lnk.Write(path, lnk.Shortcut{Target: exe, Arguments: arg, Icon: exe + ",0"}); err != nil {
		return "", err
	}
	return path, nil
}

// freeLinkPath is dir/<base>.lnk, or "<base> (2).lnk" and so on when that file is another profile's shortcut: file
// names drop characters, so distinct profile names can collide. The shortcut already made for arg is reused.
func freeLinkPath(dir, base, arg string) (string, error) {
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			name = fmt.Sprintf("%s (%d)", base, n)
		}
		path := filepath.Join(dir, name+".lnk")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return path, nil
		} else if err != nil {
			return "", err
		}
		if got, err := lnk.Arguments(path); err == nil && got == arg {
			return path, nil
		}
	}
}

func startMenuDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userdirs.AppData(home), "Microsoft", "Windows", "Start Menu", "Programs", "Mortar"), nil
}

func matchingLinks(dir, arg string) ([]string, error) {
	paths, err := lnk.Files(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, path := range paths {
		got, err := lnk.Arguments(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if got == arg {
			matches = append(matches, path)
		}
	}
	return matches, nil
}

// Renamed renames the Start menu shortcut for a profile whose name changed.
func Renamed(game, profile, profileName, gameName string) error {
	dir, matches, err := profileLinks(game, profile)
	if err != nil || len(matches) == 0 {
		return err
	}
	target := filepath.Join(dir, fileName(profileName+" ("+gameName+")")+".lnk")
	if strings.EqualFold(matches[0], target) {
		return nil
	}
	return safefs.Rename(matches[0], target)
}

// Removed removes the Start menu shortcuts for a deleted profile.
func Removed(game, profile string) error {
	_, matches, err := profileLinks(game, profile)
	if err != nil {
		return err
	}
	for _, path := range matches {
		if err := safefs.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func exists(game, profile string) (bool, error) {
	_, matches, err := profileLinks(game, profile)
	return len(matches) > 0, err
}

// profileLinks are the Start menu folder and the shortcuts in it that play the profile.
func profileLinks(game, profile string) (dir string, matches []string, err error) {
	if !validID(game) || !validID(profile) {
		return "", nil, errors.New("a shortcut needs a game and a profile")
	}
	if dir, err = startMenuDir(); err != nil {
		return "", nil, err
	}
	matches, err = matchingLinks(dir, Arg(game, profile))
	return dir, matches, err
}

// fileName keeps a shortcut's name to characters every file system accepts.
func fileName(name string) string {
	if clean := winname.Clean(name); clean != "" {
		return clean
	}
	return "Mortar profile"
}

// RemoveStartMenu deletes the Start menu's Mortar folder with every profile shortcut in it.
func RemoveStartMenu() error {
	dir, err := startMenuDir()
	if err != nil {
		return err
	}
	return safefs.RemoveAll(dir)
}

// Repoint retargets each profile shortcut in the Start menu whose program no longer exists to exe, so shortcuts
// survive a portable Mortar moving. A shortcut whose program still exists belongs to that copy and is left alone.
func Repoint(exe string) error {
	dir, err := startMenuDir()
	if err != nil {
		return err
	}
	paths, err := lnk.Files(dir)
	if err != nil {
		return err
	}
	var errs []error
	for _, path := range paths {
		arg, err := lnk.Arguments(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		target, err := lnk.Target(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !strings.HasPrefix(arg, playFlag) || strings.EqualFold(target, exe) {
			continue
		}
		if _, err := os.Stat(target); err == nil {
			continue
		}
		errs = append(errs, lnk.Write(path, lnk.Shortcut{Target: exe, Arguments: arg, Icon: exe + ",0"}))
	}
	return errors.Join(errs...)
}
