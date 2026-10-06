package datadir

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// checkOpenable refuses a path the system handler should not be given: one that is not absolute (a leading dash would
// read as an option, and a relative name depends on the working folder), a network share path, or a symbolic link to
// a file, which would open whatever it points at. A link to a folder is how a data folder moved elsewhere looks, so it
// opens.
func checkOpenable(path string) error {
	if !filepath.IsAbs(path) || isUNC(path) {
		return errors.New("not a local absolute path")
	}
	link, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if link.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Stat(path); err != nil || !target.IsDir() {
			return errors.New("a link to a file is not opened")
		}
	}
	return nil
}

// Open shows dir, a folder or a file Mortar itself owns, in the system file manager without waiting for it. The path
// is never remote data: callers build it from the data folder, a profile or the game's install.
func Open(dir string) error {
	if err := checkOpenable(dir); err != nil {
		return err
	}
	name := "xdg-open"
	switch runtime.GOOS {
	case "windows":
		name = "explorer"
	}
	cmd := exec.CommandContext(context.Background(), name, dir)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// isUNC reports a network share path, in either the plain or the \\?\UNC\ spelling.
func isUNC(path string) bool {
	if strings.HasPrefix(strings.ToUpper(path), `\\?\UNC\`) {
		return true
	}
	return strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, `\\?\`)
}
