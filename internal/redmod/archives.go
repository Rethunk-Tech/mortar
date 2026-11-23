package redmod

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// ArchiveModListPath is the file the game reads in archive/pc/mod: the archives it names load first, in that order,
// then the others (Vortex's cyberpunk2077_ext_redux load_order.archive.ts).
const ArchiveModListPath = "archive/pc/mod/modlist.txt"

// ArchiveModList is the modlist for archives in load order, or nil when the game's own order (by name) already
// loads them that way and the file is not needed.
func ArchiveModList(names []string) []string {
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			return names
		}
	}
	return nil
}

// WriteArchiveModList writes the modlist for archives in load order into the install, or removes it when the default
// order needs none.
func WriteArchiveModList(installDir string, names []string) error {
	path := filepath.Join(installDir, filepath.FromSlash(ArchiveModListPath))
	list := ArchiveModList(names)
	if list == nil {
		if err := safefs.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return safefs.WriteFileAtomic(path, []byte(strings.Join(list, "\r\n")), 0o644)
}
