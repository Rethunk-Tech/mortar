package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// manifestVersion matches a manifest's own "Version" value; dependency entries use MinimumVersion, so the first
// match is the mod's.
var manifestVersion = regexp.MustCompile(`(?i)("version"\s*:\s*")([^"]*)(")`)

// FixStaleManifest sets the Version of the uniqueID manifest inside the entry key to version, in the store item and
// in the profile's copy, when the download is confirmed to be that version but its author did not bump the
// manifest. Only the value changes, so comments and formatting survive. A manifest that came from another file of
// the entry, or a FOMOD layout, is left alone: the download's version says nothing about it.
func (s *Store) FixStaleManifest(game, id, key string, uniqueID mod.ID, version string) error {
	modsDir, err := s.ModsDir(game, id)
	if err != nil {
		return err
	}
	// The profile's copy is what SMAPI reads; the store item is fixed too when the store holds it whole, so the next
	// profile that installs it starts right.
	storeRoot, storeErr := s.items.Path(game, key)
	_, err = s.update(game, id, func(p *Profile, _ string) error {
		ei := entryIndex(p.Entries, key)
		if ei < 0 {
			return fmt.Errorf("no entry %s", key)
		}
		e := &p.Entries[ei]
		if len(e.Fomod) > 0 {
			return errNotFixable
		}
		mi := -1
		for i, m := range e.Mods {
			if mod.Equal(m.ID, uniqueID) {
				mi = i
			}
		}
		if mi < 0 {
			return errNotFixable
		}
		first, _, _ := strings.Cut(e.Mods[mi].Folder, "/")
		if slices.Contains(e.ExtraStoreKeys, first) {
			return errNotFixable
		}
		rel := filepath.Join(filepath.FromSlash(e.Mods[mi].Folder), manifest.FileName)
		if err := setManifestVersion(filepath.Join(modsDir, key, rel), uniqueID, version); err != nil {
			return err
		}
		if storeErr == nil {
			if err := setManifestVersion(filepath.Join(storeRoot, rel), uniqueID, version); err != nil && !errors.Is(err, errNotFixable) {
				return err
			}
		}
		e.Mods[mi].Version = version
		return nil
	})
	if errors.Is(err, errNotFixable) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.refreshSnapshotKey(game, id, key)
}

var errNotFixable = errors.New("manifest not in the entry's own download")

// setManifestVersion rewrites the file only when it is uniqueID's manifest; a missing file means the folder came
// from another file of the entry.
func setManifestVersion(path string, uniqueID mod.ID, version string) error {
	b, err := fsx.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return errNotFixable
	}
	if err != nil {
		return err
	}
	m, err := manifest.Parse(b)
	if err != nil || !mod.Equal(m.ModID(), uniqueID) {
		return errNotFixable
	}
	if m.Version == version {
		return nil
	}
	loc := manifestVersion.FindSubmatchIndex(b)
	if loc == nil {
		return errNotFixable
	}
	out := make([]byte, 0, len(b)+len(version))
	out = append(out, b[:loc[4]]...)
	out = append(out, strings.ReplaceAll(version, `"`, ``)...)
	out = append(out, b[loc[5]:]...)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	// Writing a new file breaks any reflink to the store copy, which is rewritten the same way.
	return fsx.WriteFile(path, out, info.Mode().Perm())
}
