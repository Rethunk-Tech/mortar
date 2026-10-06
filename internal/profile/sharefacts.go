package profile

import (
	"path/filepath"
	"regexp"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// ShareFacts are what a share link tells its viewer about a profile beyond its mods: the game version it was shared
// from, and by entry key the entry's size on disk in KiB and the newest MinimumGameVersion its mods declare.
type ShareFacts struct {
	GameVersion string
	SizeKB      map[string]int64
	MinGame     map[string]string
}

var leadingVersion = regexp.MustCompile(`^\d+(\.\d+){0,3}`)

// ShareFacts gathers the facts for p from the store's recorded sizes and the mods' manifests; what cannot be read is
// left out.
func (s *Store) ShareFacts(game string, p Profile) ShareFacts {
	out := ShareFacts{
		GameVersion: leadingVersion.FindString(s.installedGameVersion(game)),
		SizeKB:      map[string]int64{},
		MinGame:     map[string]string{},
	}
	if all, err := s.items.Entries(); err == nil {
		for _, it := range all {
			if it.Game == game && it.Size > 0 {
				out.SizeKB[it.Key] = (it.Size + 1023) / 1024
			}
		}
	}
	dir, err := s.ProfileDir(game, p.ID)
	if err != nil {
		return out
	}
	modsDir := filepath.Join(dir, "mods")
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			plain, dotted, err := ModPaths(modsDir, e.Key, m.Folder)
			if err != nil {
				continue
			}
			b, err := fsx.ReadFile(filepath.Join(plain, manifest.FileName))
			if err != nil {
				b, err = fsx.ReadFile(filepath.Join(dotted, manifest.FileName))
			}
			if err != nil {
				continue
			}
			mf, err := manifest.Parse(b)
			v := leadingVersion.FindString(mf.MinimumGameVersion)
			if err != nil || v == "" {
				continue
			}
			if c, ok := meta.CompareVersions(v, out.MinGame[e.Key]); out.MinGame[e.Key] == "" || ok && c > 0 {
				out.MinGame[e.Key] = v
			}
		}
	}
	return out
}
