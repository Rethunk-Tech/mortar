//go:build windows

package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
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
	path := filepath.Join(dir, fileName(name)+".lnk")
	if err := writeLink(path, exe, arg); err != nil {
		return "", err
	}
	return path, nil
}

func startMenuDir() (string, error) {
	appData, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Mortar"), nil
}

func withLink(path string, fn func(*ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE means COM was already initialised on this thread, which is fine.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	linkVar, err := oleutil.CallMethod(shell, "CreateShortcut", path)
	if err != nil {
		return err
	}
	link := linkVar.ToIDispatch()
	defer link.Release()
	return fn(link)
}

// writeLink drives WScript.Shell's CreateShortcut, which COM needs on one OS thread for the whole call.
func writeLink(path, exe, arg string) error {
	return withLink(path, func(link *ole.IDispatch) error {
		for prop, value := range map[string]string{"TargetPath": exe, "Arguments": arg, "IconLocation": exe + ",0"} {
			if _, err := oleutil.PutProperty(link, prop, value); err != nil {
				return err
			}
		}
		_, err := oleutil.CallMethod(link, "Save")
		return err
	})
}

func linkArguments(path string) (string, error) {
	var arguments string
	err := withLink(path, func(link *ole.IDispatch) error {
		value, err := oleutil.GetProperty(link, "Arguments")
		if err != nil {
			return err
		}
		defer func() { _ = value.Clear() }()
		arguments = value.ToString()
		return nil
	})
	return arguments, err
}

func matchingLinks(dir, arg string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".lnk") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		got, err := linkArguments(path)
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
	if !validID(game) || !validID(profile) {
		return errors.New("a shortcut needs a game and a profile")
	}
	dir, err := startMenuDir()
	if err != nil {
		return err
	}
	matches, err := matchingLinks(dir, Arg(game, profile))
	if err != nil || len(matches) == 0 {
		return err
	}
	target := filepath.Join(dir, fileName(profileName+" ("+gameName+")")+".lnk")
	if strings.EqualFold(matches[0], target) {
		return nil
	}
	return os.Rename(matches[0], target)
}

// Removed removes the Start menu shortcuts for a deleted profile.
func Removed(game, profile string) error {
	if !validID(game) || !validID(profile) {
		return errors.New("a shortcut needs a game and a profile")
	}
	dir, err := startMenuDir()
	if err != nil {
		return err
	}
	matches, err := matchingLinks(dir, Arg(game, profile))
	if err != nil {
		return err
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func exists(game, profile string) (bool, error) {
	if !validID(game) || !validID(profile) {
		return false, errors.New("a shortcut needs a game and a profile")
	}
	dir, err := startMenuDir()
	if err != nil {
		return false, err
	}
	matches, err := matchingLinks(dir, Arg(game, profile))
	return len(matches) > 0, err
}

// fileName keeps a shortcut's name to characters every file system accepts.
func fileName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < ' ' {
			return '-'
		}
		return r
	}, name)
	clean = strings.Trim(clean, " .")
	if clean == "" {
		return "Mortar profile"
	}
	return clean
}
