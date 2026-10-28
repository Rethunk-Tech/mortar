package savessvc

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func enabledPlayed(p profile.Profile) []PlayedMod {
	out := []PlayedMod{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			if !e.Enabled(m.ID) {
				continue
			}
			out = append(out, PlayedMod{
				ID: m.ID, Name: m.Name, Version: m.Version, Key: e.Key,
				SourceKind: e.Source.Kind, ModID: e.Source.ModID, FileID: e.Source.FileID,
				Repo: e.Source.Repo, Tag: e.Source.Tag, Asset: e.Source.Asset,
				ContentPackFor: m.ContentPackFor,
			})
		}
	}
	return out
}

func snapshotEnabled(store *profile.Store, game, id string) []PlayedMod {
	if store == nil || game == "" || id == "" {
		return nil
	}
	var out []PlayedMod
	_ = store.InMods(game, id, func(p profile.Profile, _ string) error {
		out = enabledPlayed(p)
		return nil
	})
	return out
}

// contentFirst puts content packs ahead of code mods, then name.
func contentFirst(a, b PlayedMod) int {
	ac, bc := a.ContentPackFor != "", b.ContentPackFor != ""
	if ac != bc {
		if ac {
			return -1
		}
		return 1
	}
	return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
}

func sortRecorded(mods []PlayedMod) []PlayedMod {
	out := slices.Clone(mods)
	slices.SortFunc(out, contentFirst)
	return out
}

// MissingFrom is the recorded list the profile lacks or has switched off. Content packs come first.
func MissingFrom(recorded []PlayedMod, present, enabled map[string]bool) []Lack {
	out := []Lack{}
	seen := map[string]bool{}
	for _, m := range sortRecorded(recorded) {
		id := m.ID.Fold()
		if id == "" || seen[id] || enabled[id] {
			continue
		}
		seen[id] = true
		name := m.Name
		name = cmp.Or(name, m.ID.Local())
		out = append(out, Lack{ID: m.ID, Name: name, Disabled: present[id]})
	}
	return out
}

func haveMaps(mods []profile.Mod) (present, enabled map[string]bool) {
	present, enabled = map[string]bool{}, map[string]bool{}
	for _, m := range mods {
		id := m.ID.Fold()
		present[id] = true
		enabled[id] = enabled[id] || m.Enabled
	}
	return present, enabled
}

func profileExists(store *profile.Store, game, id string) bool {
	if store == nil || id == "" {
		return false
	}
	// A damaged profile still exists.
	_, err := store.Get(game, id)
	return err == nil || usererr.KindOf(err) == usererr.Damaged
}

func (s *Service) fillLast(game string, fit *Fit, present, enabled map[string]bool) {
	if s.last == nil || fit.Folder == "" {
		return
	}
	rec, ok, err := s.last.Get(game, fit.Folder)
	if err != nil || !ok {
		return
	}
	fit.LastProfileID = rec.ProfileID
	fit.LastProfileAt = rec.At.UnixMilli()
	fit.LastMods = rec.Mods
	fit.LastProfileExists = profileExists(s.profiles, game, rec.ProfileID)
	if present != nil {
		fit.LastMissing = MissingFrom(rec.Mods, present, enabled)
	}
}

// NotePlayed records that profileID just ran saveFolder, with the mods that profile had on.
//
//wails:ignore
func (s *Service) NotePlayed(gameID, profileID, saveFolder string) {
	if s.last == nil {
		return
	}
	_ = s.last.RecordRun(gameID, saveFolder, profileID, time.Now(), snapshotEnabled(s.profiles, gameID, profileID))
}

// asProfileIDs names each mod a save records by its mod-root id (modinfo:<id>) as the profile's mod with that id, so
// it matches, enables and dismisses as that mod; a mod the profile lacks keeps the save's name.
func asProfileIDs(used []mod.ID, mods []profile.Mod) []mod.ID {
	byModID := map[string]mod.ID{}
	for _, m := range mods {
		if m.ModID != "" {
			byModID[strings.ToLower(m.ModID)] = m.ID
		}
	}
	if len(byModID) == 0 {
		return used
	}
	out := make([]mod.ID, len(used))
	for i, id := range used {
		out[i] = id
		if id.Format() == mod.FormatModInfo {
			if own, ok := byModID[strings.ToLower(id.Local())]; ok {
				out[i] = own
			}
		}
	}
	return out
}
