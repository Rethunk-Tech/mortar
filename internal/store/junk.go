package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func junkDir(name string) bool {
	return strings.EqualFold(name, "__MACOSX") || name == ".git"
}

func thumbsOnly(dir string) bool {
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) == 0 {
		return false
	}
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(e.Name(), "Thumbs.db") {
			return false
		}
	}
	return true
}

func stripJunk(root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root || !d.IsDir() {
			return err
		}
		if junkDir(d.Name()) {
			_ = os.RemoveAll(p)
			return fs.SkipDir
		}
		return nil
	})
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	slices.SortFunc(dirs, func(a, b string) int { return len(b) - len(a) })
	for _, dir := range dirs {
		if dir != root && thumbsOnly(dir) {
			_ = os.RemoveAll(dir)
		}
	}
}
