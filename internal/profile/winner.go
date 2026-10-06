package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// SetWinner records or clears that winner, a pack of the download winnerKey, loads after loser, and rewrites that
// pack's installed manifest so SMAPI loads it later. The download's other packs are left alone.
func (s *Store) SetWinner(game, profileID, winnerKey string, winner, loser mod.ID, on bool) (Profile, error) {
	if loser == "" {
		return Profile{}, fmt.Errorf("loser unique ID is empty")
	}
	p, err := s.updateMods(game, profileID, func(p *Profile, dir string) error {
		i := entryIndex(p.Entries, winnerKey)
		if i < 0 {
			return fmt.Errorf("mod %q is not in this profile", winnerKey)
		}
		e := p.Entries[i]
		ci := slices.IndexFunc(e.Mods, func(c Component) bool { return mod.Equal(c.ID, winner) })
		if ci < 0 {
			return fmt.Errorf("%s is not in %q", winner, winnerKey)
		}
		if on {
			if err := loserNeedsWinner(p.Entries, e, e.Mods[ci], loser); err != nil {
				return err
			}
		}
		e.Mods = slices.Clone(e.Mods)
		e.Mods[ci].LoadAfter = setLoadAfter(e.Mods[ci].LoadAfter, loser, on)
		p.Entries[i] = e
		var dropped []mod.ID
		if !on && !authorDeclares(storePeer(s, game, e.Key), e.Mods[ci].Folder, loser) {
			dropped = []mod.ID{loser}
		}
		return rewriteLoadAfter(filepath.Join(dir, "mods", e.Key), e, e.Mods[ci], dropped)
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.refreshSnapshotKey(game, profileID, winnerKey)
}

// SetWinner records or clears that winner, a pack of the download winnerKey, loads after loser.
func (s *Service) SetWinner(game, profileID, winnerKey string, winner, loser mod.ID, on bool) (Profile, error) {
	return s.store.SetWinner(game, profileID, winnerKey, winner, loser, on)
}

func entryIndex(entries []Entry, key string) int {
	return slices.IndexFunc(entries, func(e Entry) bool { return e.Key == key })
}

func requireEntry(entries []Entry, key string) (int, error) {
	i := entryIndex(entries, key)
	if i < 0 {
		return 0, fmt.Errorf("%q is not in this profile", key)
	}
	return i, nil
}

func setLoadAfter(ids []mod.ID, loser mod.ID, on bool) []mod.ID {
	out := make([]mod.ID, 0, len(ids)+1)
	for _, id := range ids {
		if !mod.Equal(id, loser) {
			out = append(out, id)
		}
	}
	if on {
		out = append(out, loser)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// applyLoadAfter writes every pack's recorded wins into its manifest under root.
func applyLoadAfter(root string, e Entry) error {
	for _, m := range e.Mods {
		if len(m.LoadAfter) == 0 {
			continue
		}
		if err := rewriteLoadAfter(root, e, m, nil); err != nil {
			return err
		}
	}
	return nil
}

func rewriteLoadAfter(root string, e Entry, m Component, dropped []mod.ID) error {
	path := componentManifest(root, m.Folder)
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	rewritten, err := manifest.RewriteDependencies(raw, safeLoadAfter(e, m), dropped)
	if err != nil {
		return err
	}
	return fsx.WriteFile(path, rewritten, 0o644)
}

func componentManifest(root, folder string) string {
	rel := filepath.FromSlash(folder)
	if folder == "." {
		rel = ""
	}
	return filepath.Join(root, rel, manifest.FileName)
}

// keepLoadAfter carries the recorded wins of prev onto the rebuilt packs with the same id.
func keepLoadAfter(rebuilt, prev []Component) {
	for i := range rebuilt {
		if j := slices.IndexFunc(prev, func(c Component) bool { return mod.Equal(c.ID, rebuilt[i].ID) }); j >= 0 {
			rebuilt[i].LoadAfter = prev[j].LoadAfter
		}
	}
}

// safeLoadAfter is a component's LoadAfter as it may declare it: never itself, and never a component of
// the same download that needs it, since SMAPI refuses to load a dependency cycle.
func safeLoadAfter(e Entry, m Component) []mod.ID {
	return slices.DeleteFunc(slices.Clone(m.LoadAfter), func(id mod.ID) bool {
		if mod.Equal(id, m.ID) {
			return true
		}
		i := slices.IndexFunc(e.Mods, func(c Component) bool { return mod.Equal(c.ID, id) })
		return i >= 0 && slices.ContainsFunc(e.Mods[i].Needs, func(n mod.ID) bool { return mod.Equal(n, m.ID) })
	})
}

// loserNeedsWinner refuses an order SMAPI could never honour: a loser in another download that needs the winner
// always loads after it.
func loserNeedsWinner(entries []Entry, winnerEntry Entry, winner Component, loser mod.ID) error {
	for _, e := range entries {
		if e.Key == winnerEntry.Key {
			continue
		}
		for _, c := range e.Mods {
			if !mod.Equal(c.ID, loser) {
				continue
			}
			if slices.ContainsFunc(c.Needs, func(n mod.ID) bool { return mod.Equal(n, winner.ID) }) {
				return fmt.Errorf("%s needs %s, so it always loads after it", c.Name, winner.Name)
			}
		}
	}
	return nil
}

// authorDeclares reports whether the store's copy of a component already lists id, so undoing a win keeps it.
func authorDeclares(peer, folder string, id mod.ID) bool {
	if peer == "" {
		return false
	}
	raw, err := fsx.ReadFile(filepath.Join(peer, filepath.FromSlash(folder), manifest.FileName))
	if err != nil {
		return false
	}
	m, err := manifest.Parse(raw)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(m.Dependencies, func(d manifest.Dependency) bool { return mod.Equal(mod.SMAPI(d.UniqueID), id) })
}

// adoptProfileLoadAfter is adoptEntryLoadAfter for a whole profile file.
func adoptProfileLoadAfter(raw []byte, p *Profile, modsDir string) {
	if !bytes.Contains(raw, loadAfterKey) {
		return
	}
	var doc struct {
		Entries json.RawMessage `json:"entries"`
	}
	if json.Unmarshal(raw, &doc) == nil {
		adoptEntryLoadAfter(doc.Entries, p.Entries, modsDir)
	}
}

var loadAfterKey = []byte(`"loadAfter"`)

// adoptEntryLoadAfter moves wins that rawEntries, the JSON entries decoded, still records per download onto the
// packs whose manifest lists the loser: that is the only trace of which pack the user meant.
func adoptEntryLoadAfter(rawEntries []byte, entries []Entry, modsDir string) {
	if !bytes.Contains(rawEntries, loadAfterKey) {
		return
	}
	var old []struct {
		LoadAfter []mod.ID `json:"loadAfter"`
	}
	if json.Unmarshal(rawEntries, &old) != nil || len(old) != len(entries) {
		return
	}
	for i, oe := range old {
		e := &entries[i]
		for _, loser := range oe.LoadAfter {
			for ci := range e.Mods {
				if manifestLists(modsDir, e.Key, e.Mods[ci].Folder, loser) {
					e.Mods[ci].LoadAfter = setLoadAfter(e.Mods[ci].LoadAfter, loser, true)
				}
			}
		}
	}
}

func manifestLists(modsDir, key, folder string, id mod.ID) bool {
	plain, dotted, err := ModPaths(modsDir, key, folder)
	if err != nil {
		return false
	}
	for _, dir := range []string{plain, dotted} {
		raw, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName))
		if err != nil {
			continue
		}
		m, err := manifest.Parse(raw)
		return err == nil && slices.ContainsFunc(m.Dependencies, func(d manifest.Dependency) bool { return mod.Equal(d.ModID(), id) })
	}
	return false
}
