package configsvc

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// ModFile is one config file of a mod in the Config page's list.
type ModFile struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	// Changed is true when a setting of the file differs from its default.
	Changed bool `json:"changed"`
}

// ModConfig is one mod with a config source.
type ModConfig struct {
	ID      mod.ID    `json:"id"`
	Key     string    `json:"key"`
	Name    string    `json:"name"`
	Enabled bool      `json:"enabled"`
	Files   []ModFile `json:"files"`
	// HasMenu is true when the bridge captured the mod's in-game settings menu.
	HasMenu bool `json:"hasMenu"`
	// Pending counts the in-game menu edits that wait for the game's next start.
	Pending int `json:"pending"`
}

// ModList is everything the Config page lists for a profile.
type ModList struct {
	Mods []ModConfig `json:"mods"`
	// Other holds the .cfg files no mod owns, such as BepInEx.cfg.
	Other []ModFile `json:"other"`
	// Without counts the profile's mods that have no config source yet: mods write theirs the first time the game runs.
	Without int `json:"without"`
}

// Mods lists the profile's mods that have a config source (a SMAPI config.json, a BepInEx .cfg owned through its
// plugin GUID, a captured in-game menu) in one pass, with the .cfg files nobody owns.
func (s *Service) Mods(game, profileID string) (ModList, error) {
	out := ModList{Mods: []ModConfig{}, Other: []ModFile{}}
	mods, err := s.Profiles.UserMods(game, profileID)
	if err != nil {
		return out, err
	}
	docs, err := s.cfgDocs(game, profileID)
	if err != nil {
		return out, err
	}
	view, err := s.Profiles.ConfigView(game, profileID)
	if err != nil {
		return out, err
	}
	dir, _ := s.Profiles.ProfileDir(game, profileID)
	owned := map[string]bool{}
	for _, m := range mods {
		id := string(m.ID)
		row := ModConfig{ID: m.ID, Key: m.Key, Name: m.Name, Enabled: m.Enabled, Files: []ModFile{}}
		if folder, err := view.ModFolder("", m.ID); err == nil {
			if _, err := os.Stat(filepath.Join(folder, jsonName)); err == nil {
				row.Files = append(row.Files, ModFile{Name: jsonName, Format: FormatSMAPI, Changed: s.jsonChanged(view, m.Key, m.ID, folder)})
			}
		}
		if dir != "" {
			if _, err := os.Stat(gmcm.CapturePath(dir, m.ID)); err == nil {
				row.HasMenu = true
				if p, err := gmcm.ReadPending(dir, m.ID); err == nil {
					row.Pending = len(p.Edits)
				}
				row.Files = append(row.Files, ModFile{Name: gmcmFileName, Format: FormatGMCM})
			}
		}
		names := append([]string{id}, view.PluginGUIDs(m.ID)...)
		for _, d := range docs {
			if d.ownedBy(names) {
				owned[d.name] = true
				row.Files = append(row.Files, ModFile{Name: d.name, Format: FormatBepInEx, Changed: schemaChanged(d.doc.schema(d.file()))})
			}
		}
		if len(row.Files) == 0 {
			out.Without++
			continue
		}
		out.Mods = append(out.Mods, row)
	}
	for _, d := range docs {
		if !owned[d.name] {
			out.Other = append(out.Other, ModFile{Name: d.name, Format: FormatBepInEx, Changed: schemaChanged(d.doc.schema(d.file()))})
		}
	}
	return out, nil
}

// changedKey identifies one result of jsonChanged: the shipped config of an entry never changes under its key, so the
// current file's stat is all that can invalidate it.
type changedKey struct {
	path, key string
	size      int64
	mtime     int64
}

const changedCacheMax = 4096

// jsonChanged reports whether folder's config.json differs from the shipped one, remembering the answer until the file
// changes.
func (s *Service) jsonChanged(view profile.ConfigView, key string, id mod.ID, folder string) bool {
	path := filepath.Join(folder, jsonName)
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	k := changedKey{path: path, key: key, size: fi.Size(), mtime: fi.ModTime().UnixNano()}
	s.changedMu.Lock()
	v, ok := s.changed[k]
	s.changedMu.Unlock()
	if ok {
		return v
	}
	v = jsonChanged(view, id)
	s.changedMu.Lock()
	if s.changed == nil || len(s.changed) >= changedCacheMax {
		s.changed = map[changedKey]bool{}
	}
	s.changed[k] = v
	s.changedMu.Unlock()
	return v
}

func jsonChanged(view profile.ConfigView, id mod.ID) bool {
	cur, err := view.ReadConfig("", id)
	if err != nil {
		return false
	}
	shipped, _ := view.ShippedConfig("", id)
	sc, err := jsonSchema(ConfigFile{Name: jsonName, Format: FormatSMAPI, Label: jsonName}, cur, shipped, nil, nil)
	return err == nil && schemaChanged(sc)
}

func schemaChanged(sc Schema) bool {
	for _, sec := range sc.Sections {
		for _, e := range sec.Entries {
			if e.HasDefault && e.Value != e.Default {
				return true
			}
		}
	}
	return false
}
