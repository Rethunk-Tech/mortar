package profile

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Installed is one mod of a profile with its manifest as it stands in mods/.
type Installed struct {
	Key           string
	Folder        string
	Source        Source
	Enabled       bool
	Pinned        bool
	SkipVersion   string
	SkipSources   []string
	IgnoreUpdates bool
	UpdateChannel string
	LoadAfter     []mod.ID
	manifest.Manifest
}

// Installed reads the manifest of every mod in the profile. A mod whose folder or manifest is gone or no longer
// parses is skipped, since nothing can be said about it. A package has no SMAPI manifest, so its components stand
// as recorded at install, with the store item as their folder.
func (s *Store) Installed(game, id string) ([]Installed, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	var out []Installed
	for _, e := range p.Entries {
		if !e.hasFolder() {
			out = append(out, s.packageInstalled(game, e)...)
			continue
		}
		for _, m := range e.Mods {
			plain, dotted, err := ModPaths(filepath.Join(dir, "mods"), e.Key, m.Folder)
			if err != nil {
				return nil, err
			}
			enabled := e.Enabled(m.ID)
			folder := plain
			if !enabled {
				folder = dotted
			}
			b, err := fsx.ReadFile(filepath.Join(folder, manifest.FileName))
			if err != nil {
				continue
			}
			mf, err := manifest.Parse(b)
			if err != nil {
				continue
			}
			out = append(out, Installed{
				Key: e.Key, Folder: folder, Source: e.Source, Enabled: enabled,
				Pinned: e.Pinned, SkipVersion: e.SkipVersion, SkipSources: e.SkipSources, IgnoreUpdates: e.IgnoreUpdates,
				UpdateChannel: e.UpdateChannel,
				LoadAfter:     e.LoadAfter, Manifest: mf,
			})
		}
	}
	return out, nil
}

func (s *Store) packageInstalled(game string, e Entry) []Installed {
	var dir string
	if s.items != nil {
		dir, _ = s.items.Path(game, e.Key)
	}
	out := make([]Installed, 0, len(e.Mods))
	for _, c := range e.Mods {
		out = append(out, Installed{
			Key: e.Key, Folder: dir, Source: e.Source, Enabled: e.Enabled(c.ID),
			Pinned: e.Pinned, SkipVersion: e.SkipVersion, SkipSources: e.SkipSources, IgnoreUpdates: e.IgnoreUpdates,
			UpdateChannel: e.UpdateChannel, LoadAfter: e.LoadAfter,
			Manifest: manifest.Manifest{Name: c.Name, Author: c.Author, Version: c.Version}.WithModID(c.ID),
		})
	}
	return out
}
