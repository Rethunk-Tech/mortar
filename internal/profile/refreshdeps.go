package profile

import (
	"log"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

// RefreshDependencies re-reads every entry's manifests from the store so the stored needs and optional lists
// follow the current manifest parser. It writes a profile only when something changed, and records no history.
func (s *Store) RefreshDependencies(game string) error {
	profiles, err := s.listOK(game)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err := s.refreshDependencies(game, p.ID); err != nil {
			log.Printf("profile %s/%s: refresh dependencies: %v", game, p.ID, err)
		}
	}
	return nil
}

func (s *Store) refreshDependencies(game, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	changed := false
	for ei := range p.Entries {
		e := &p.Entries[ei]
		peer := storePeer(s, game, e.Key)
		if peer == "" {
			continue
		}
		found, err := manifest.Scan(peer)
		if err != nil {
			continue
		}
		fresh := map[string]Component{}
		for _, m := range entryMods(found) {
			fresh[m.ID.Fold()] = m
		}
		for mi := range e.Mods {
			m := &e.Mods[mi]
			f, ok := fresh[m.ID.Fold()]
			if !ok || (slices.Equal(m.Needs, f.Needs) && slices.Equal(m.Optional, f.Optional)) {
				continue
			}
			m.Needs, m.Optional = f.Needs, f.Optional
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return writeProfile(dir, p)
}
