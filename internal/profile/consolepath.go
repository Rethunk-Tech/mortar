package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

// ConsoleRevealDir returns the folder to open for a log path that stays inside one of the roots.
func ConsoleRevealDir(p string, roots []string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(filepath.Clean(filepath.FromSlash(p)))
	if err != nil {
		return "", err
	}
	target := abs
	if resolved, err := fsx.EvalSymlinks(abs); err == nil {
		target = resolved
	} else if resolved, err := fsx.EvalSymlinks(filepath.Dir(abs)); err == nil {
		target = filepath.Join(resolved, filepath.Base(abs))
	}
	dir := target
	info, err := os.Lstat(target)
	if err != nil || !info.IsDir() {
		dir = filepath.Dir(target)
	}
	if resolved, err := fsx.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		r := filepath.Clean(root)
		if resolved, err := fsx.EvalSymlinks(r); err == nil {
			r = resolved
		}
		if datadir.UnderRoot(r, dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("path is outside the profile mods folder and the game folder")
}

// ModsDir returns the profile's mods folder.
func (s *Service) ModsDir(gameID, id string) (string, error) {
	return s.store.ModsDir(gameID, id)
}

// OpenConsolePath opens the containing folder of path when it is inside the profile's mods folder or the game folder.
func (s *Service) OpenConsolePath(gameID, id, path string) error {
	mods, err := s.store.ModsDir(gameID, id)
	if err != nil {
		return err
	}
	roots := []string{mods}
	if s.settings != nil {
		cur := s.settings.Get()
		if dir, err := game.InstallDir(s.home, cur, gameID); err == nil && dir != "" {
			roots = append(roots, dir)
		}
	}
	dir, err := ConsoleRevealDir(path, roots)
	if err != nil {
		return err
	}
	return datadir.Open(dir)
}
