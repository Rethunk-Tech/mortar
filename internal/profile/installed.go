package profile

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
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
	Implicit      bool
	SkipVersion   string
	SkipSources   []string
	IgnoreUpdates bool
	UpdateChannel string
	LoadAfter     []mod.ID
	manifest.Manifest
}

// UnreadableManifest is a mod of the profile whose manifest is gone (Missing) or no longer parses.
type UnreadableManifest struct {
	Key     string
	Folder  string
	Missing bool
}

// Installed reads the manifest of every mod in the profile. A mod whose manifest is gone or no longer parses is left
// out, since nothing can be said about it; UnreadableManifests names those. A package has no SMAPI manifest, so its
// components stand as recorded at install, with the store item as their folder and the needs its Thunderstore
// manifest lists.
func (s *Store) Installed(game, id string) ([]Installed, error) {
	out, _, err := s.installed(game, id)
	return out, err
}

// UnreadableManifests lists the profile's mods that Installed leaves out for their manifest.
func (s *Store) UnreadableManifests(game, id string) ([]UnreadableManifest, error) {
	_, bad, err := s.installed(game, id)
	return bad, err
}

func (s *Store) installed(game, id string) ([]Installed, []UnreadableManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, nil, err
	}
	var out []Installed
	var bad []UnreadableManifest
	for _, e := range p.Entries {
		if !e.hasFolder() {
			out = append(out, s.packageInstalled(game, e)...)
			continue
		}
		for _, m := range e.Mods {
			plain, dotted, err := ModPaths(filepath.Join(dir, "mods"), e.Key, m.Folder)
			if err != nil {
				return nil, nil, err
			}
			enabled := e.Enabled(m.ID)
			folder := plain
			if !enabled {
				folder = dotted
			}
			b, err := fsx.ReadFile(filepath.Join(folder, manifest.FileName))
			if err != nil {
				// A folder that is gone altogether is drift, which offers Restore.
				if _, statErr := fsx.Stat(folder); statErr == nil {
					bad = append(bad, UnreadableManifest{Key: e.Key, Folder: folder, Missing: true})
				}
				continue
			}
			mf, err := manifest.Parse(b)
			if err != nil {
				bad = append(bad, UnreadableManifest{Key: e.Key, Folder: folder})
				continue
			}
			out = append(out, Installed{
				Key: e.Key, Folder: folder, Source: e.Source, Enabled: enabled,
				Pinned: e.Pinned, Implicit: e.Implicit, SkipVersion: e.SkipVersion, SkipSources: e.SkipSources, IgnoreUpdates: e.IgnoreUpdates,
				UpdateChannel: e.UpdateChannel,
				LoadAfter:     m.LoadAfter, Manifest: mf,
			})
		}
	}
	return out, bad, nil
}

func (s *Store) packageInstalled(game string, e Entry) []Installed {
	var dir string
	if s.items != nil {
		dir, _ = s.items.Path(game, e.StoreKey())
	}
	var needs []manifest.Dependency
	if dir != "" {
		if b, err := fsx.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
			if m, err := bepinex5.ParseManifest(b); err == nil {
				needs = m.Needs()
			}
		}
	}
	out := make([]Installed, 0, len(e.Mods))
	for _, raw := range e.Mods {
		c := e.Shown(raw)
		mf := manifest.Manifest{Name: c.Name, Author: c.Author, Version: c.Version, MinimumGameVersion: c.MinGame}.WithModID(c.ID)
		mf.Dependencies = needs
		out = append(out, Installed{
			Key: e.Key, Folder: dir, Source: e.Source, Enabled: e.Enabled(c.ID),
			Pinned: e.Pinned, Implicit: e.Implicit, SkipVersion: e.SkipVersion, SkipSources: e.SkipSources, IgnoreUpdates: e.IgnoreUpdates,
			UpdateChannel: e.UpdateChannel, LoadAfter: c.LoadAfter,
			Manifest: mf,
		})
	}
	return out
}
