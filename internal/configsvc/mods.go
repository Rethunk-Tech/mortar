package configsvc

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
	dir, _ := s.Profiles.ProfileDir(game, profileID)
	owned := map[string]bool{}
	for _, m := range mods {
		id := string(m.ID)
		row := ModConfig{ID: m.ID, Key: m.Key, Name: m.Name, Enabled: m.Enabled, Files: []ModFile{}}
		if folder, err := s.Profiles.ModFolder(game, profileID, "", m.ID); err == nil {
			if _, err := os.Stat(filepath.Join(folder, jsonName)); err == nil {
				row.Files = append(row.Files, ModFile{Name: jsonName, Format: FormatSMAPI, Changed: s.jsonChanged(game, profileID, m.ID)})
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
		names := append([]string{id}, s.Profiles.PluginGUIDs(game, profileID, m.ID)...)
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

func (s *Service) jsonChanged(game, profileID string, id mod.ID) bool {
	cur, err := s.Profiles.ReadConfig(game, profileID, "", id)
	if err != nil {
		return false
	}
	shipped, _ := s.Profiles.ShippedConfig(game, profileID, "", id)
	sc, err := jsonSchema(ConfigFile{Name: jsonName, Format: FormatSMAPI, Label: jsonName}, cur, shipped, nil)
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
