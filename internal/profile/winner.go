package profile

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// SetWinner records or clears a LoadAfter on winnerKey for loser and rewrites the winner's
// installed manifest so SMAPI loads the winner later.
func (s *Store) SetWinner(game, profileID, winnerKey string, loser mod.ID, on bool) (Profile, error) {
	if loser == "" {
		return Profile{}, fmt.Errorf("loser unique ID is empty")
	}
	p, err := s.updateMods(game, profileID, func(p *Profile, dir string) error {
		i := entryIndex(p.Entries, winnerKey)
		if i < 0 {
			return fmt.Errorf("mod %q is not in this profile", winnerKey)
		}
		e := p.Entries[i]
		if on {
			if err := loserNeedsWinner(p.Entries, e, loser); err != nil {
				return err
			}
		}
		e.LoadAfter = setLoadAfter(e.LoadAfter, loser, on)
		p.Entries[i] = e
		var drop func(Component) []mod.ID
		if !on {
			peer := storePeer(s, game, e.Key)
			drop = func(m Component) []mod.ID {
				if authorDeclares(peer, m.Folder, loser) {
					return nil
				}
				return []mod.ID{loser}
			}
		}
		return applyLoadAfter(filepath.Join(dir, "mods", e.Key), e, drop)
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.refreshSnapshotKey(game, profileID, winnerKey)
}

// SetWinner records or clears a LoadAfter on winnerKey for loser.
func (s *Service) SetWinner(game, profileID, winnerKey string, loser mod.ID, on bool) (Profile, error) {
	return s.store.SetWinner(game, profileID, winnerKey, loser, on)
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

func applyLoadAfter(root string, e Entry, drop func(Component) []mod.ID) error {
	if len(e.LoadAfter) == 0 && drop == nil {
		return nil
	}
	for _, m := range e.Mods {
		rel := filepath.FromSlash(m.Folder)
		if m.Folder == "." {
			rel = ""
		}
		path := filepath.Join(root, rel, manifest.FileName)
		raw, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		var dropped []mod.ID
		if drop != nil {
			dropped = drop(m)
		}
		rewritten, err := manifest.RewriteDependencies(raw, safeLoadAfter(e, m), dropped)
		if err != nil {
			return err
		}
		if err := fsx.WriteFile(path, rewritten, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// safeLoadAfter is the entry's LoadAfter as one component may declare it: never itself, and never a component of
// the same download that needs it, since SMAPI refuses to load a dependency cycle.
func safeLoadAfter(e Entry, m Component) []mod.ID {
	return slices.DeleteFunc(slices.Clone(e.LoadAfter), func(id mod.ID) bool {
		if mod.Equal(id, m.ID) {
			return true
		}
		i := slices.IndexFunc(e.Mods, func(c Component) bool { return mod.Equal(c.ID, id) })
		return i >= 0 && slices.ContainsFunc(e.Mods[i].Needs, func(n mod.ID) bool { return mod.Equal(n, m.ID) })
	})
}

// loserNeedsWinner refuses an order SMAPI could never honour: a loser in another download that needs the winner
// always loads after it.
func loserNeedsWinner(entries []Entry, winner Entry, loser mod.ID) error {
	for _, e := range entries {
		if e.Key == winner.Key {
			continue
		}
		for _, c := range e.Mods {
			if !mod.Equal(c.ID, loser) {
				continue
			}
			for _, w := range winner.Mods {
				if slices.ContainsFunc(c.Needs, func(n mod.ID) bool { return mod.Equal(n, w.ID) }) {
					return fmt.Errorf("%s needs %s, so it always loads after it", c.Name, w.Name)
				}
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
