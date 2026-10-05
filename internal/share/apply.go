package share

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
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
			plain, dotted, pathErr := profile.ModPaths(modsDir, e.Key, m.Folder)
			if pathErr != nil {
				continue
			}
			folder := plain
			if !fsx.IsDir(folder) {
				folder = dotted
			}
			if !fsx.IsDir(folder) {
				continue
			}
			for _, c := range configs {
				if !mod.Equal(c.ID, m.ID) || !validConfigPath(c.Path) || strings.EqualFold(c.Path, manifest.FileName) {
					continue
				}
				dest := filepath.Join(folder, filepath.FromSlash(c.Path))
				if !filepath.IsLocal(filepath.FromSlash(c.Path)) {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
					return written, err
				}
				if err := fsx.WriteFile(dest, c.Data, configPerm); err != nil {
					return written, err
				}
				if !done[m.ID.Fold()] {
					done[m.ID.Fold()] = true
					written = append(written, m.ID)
				}
			}
		}
	}
	return written, nil
}
