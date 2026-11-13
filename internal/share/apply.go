package share

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const configPerm = 0o644

// Apply writes configs into the folders of the entries' mods, matched by mod id, and returns the ids it
// wrote for. It re-checks every path, since a Config may not have come through Read: only a plain relative
// .json path is written, never a manifest, and only inside a folder the entry's mod already has, switched on or
// off. modsDir is the profile's mods/ folder.
func Apply(modsDir string, entries []profile.Entry, configs []Config) (written []mod.ID, err error) {
	done := map[string]bool{}
	for _, e := range entries {
		for _, m := range e.Mods {
			folder, ok := existingModFolder(modsDir, e.Key, m.Folder)
			if !ok {
				continue
			}
			wrote, err := applyToFolder(folder, m.ID, configs)
			if wrote && !done[m.ID.Fold()] {
				done[m.ID.Fold()] = true
				written = append(written, m.ID)
			}
			if err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

// existingModFolder returns the mod's folder on disk, switched on or off, and whether it exists.
func existingModFolder(modsDir, key, folder string) (string, bool) {
	plain, dotted, err := profile.ModPaths(modsDir, key, folder)
	if err != nil {
		return "", false
	}
	if safefs.IsDir(plain) {
		return plain, true
	}
	return dotted, safefs.IsDir(dotted)
}

// applyToFolder writes the configs of one mod id into folder and reports whether it wrote any.
func applyToFolder(folder string, id mod.ID, configs []Config) (wrote bool, err error) {
	for _, c := range configs {
		if !mod.Equal(c.ID, id) || !ValidConfigPath(c.Path) || strings.EqualFold(c.Path, manifestFile) {
			continue
		}
		dest := filepath.Join(folder, filepath.FromSlash(c.Path))
		if !filepath.IsLocal(filepath.FromSlash(c.Path)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return wrote, err
		}
		if err := safefs.WriteFileAtomic(dest, c.Data, configPerm); err != nil {
			return wrote, err
		}
		wrote = true
	}
	return wrote, nil
}
