package profile

import (
	"errors"
	"log"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// FomodRelayout counts what RelayoutFomod did: entries rebuilt by the spec's destinations, entries the spec lays out
// as before, entries left on the old rule because their archive is no longer in the store, and failures.
type FomodRelayout struct {
	Moved, Unchanged, NoArchive, Failed int
}

// RelayoutFomod moves every entry placed before Mortar followed the FOMOD spec's destinations to the spec's layout,
// rebuilt from its stored archive as an update to the same archive is, so the player's files move with their mod and
// nothing is left at the old place. A game busy reports launching or running is left alone, as is an entry whose
// archive is gone; both are tried again on the next run.
func (s *Store) RelayoutFomod(busy func(game string) bool) FomodRelayout {
	var out FomodRelayout
	dirs, _ := filepath.Glob(filepath.Join(s.root, "*"))
	for _, gameDir := range dirs {
		g := filepath.Base(gameDir)
		if !game.Valid(g) || (busy != nil && busy(g)) {
			continue
		}
		all, err := s.listOK(g)
		if err != nil {
			log.Printf("fomod relayout %s: %v", g, err)
			continue
		}
		for _, p := range all {
			for _, e := range p.Entries {
				if e.FomodSpec {
					continue
				}
				switch r, err := s.relayoutEntry(g, p.ID, e.Key); {
				case err != nil:
					out.Failed++
					log.Printf("fomod relayout %s/%s %s: %v", g, p.ID, e.Key, err)
				case r == relayoutMoved:
					out.Moved++
				case r == relayoutUnchanged:
					out.Unchanged++
				case r == relayoutNoArchive:
					out.NoArchive++
				}
			}
		}
	}
	return out
}

type relayoutResult int

const (
	relayoutUnchanged relayoutResult = iota
	relayoutMoved
	relayoutNoArchive
)

// errNoArchive leaves profile.json as it was for an entry whose archive the store no longer holds.
var errNoArchive = errors.New("archive not in the store")

func (s *Store) relayoutEntry(game, id, key string) (relayoutResult, error) {
	s.setHistoryQuiet(id, true)
	defer s.setHistoryQuiet(id, false)
	if err := s.lockUnlocked(game, id); err != nil {
		return 0, err
	}
	defer s.mu.Unlock()
	res := relayoutUnchanged
	var sw swapped
	_, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		ei, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		e := p.Entries[ei]
		if e.IsOverlay() || !e.hasFolder() {
			p.Entries[ei].FomodSpec = true
			return nil
		}
		for _, k := range append([]string{e.StoreKey()}, e.ExtraStoreKeys...) {
			if _, err := s.items.Path(game, k); errors.Is(err, store.ErrNotFound) {
				return errNoArchive
			} else if err != nil {
				return err
			}
		}
		_, legacy, driver, err := s.layoutOf(game, id, e.StoreKey(), e.Fomod, true)
		if err != nil {
			return err
		}
		if driver == "fomod" {
			_, spec, _, err := s.layoutOf(game, id, e.StoreKey(), e.Fomod, false)
			if err != nil {
				return err
			}
			if !slices.Equal(legacy.Files, spec.Files) {
				res = relayoutMoved
				sw, err = s.relayoutFolder(game, id, dir, p, ei)
				return err
			}
		}
		p.Entries[ei].FomodSpec = true
		return nil
	})
	if errors.Is(err, errNoArchive) {
		return relayoutNoArchive, nil
	}
	if err != nil {
		return 0, errors.Join(err, sw.undo())
	}
	sw.commit()
	return res, nil
}

// relayoutFolder rebuilds entry ei through the update step, keeping everything about the entry an update to another
// version would change, other than its folder and mods.
func (s *Store) relayoutFolder(game, id, dir string, p *Profile, ei int) (swapped, error) {
	e := p.Entries[ei]
	source := e.Source.WithFomod(e.Fomod)
	ne, sw, err := s.swapEntry(game, id, dir, e, e.Key, &source)
	if err != nil {
		return sw, err
	}
	ne.Source, ne.PreviousKey, ne.PreviousSource = e.Source, e.PreviousKey, e.PreviousSource
	ne.ExtraStoreKeys, ne.PreviousExtraStoreKeys = e.ExtraStoreKeys, e.PreviousExtraStoreKeys
	p.Entries[ei] = ne
	return sw, nil
}
