package profile

import (
	"errors"
	"slices"
)

// MovePackage moves the entry up (delta < 0, lower priority) or down (delta > 0, wins files) past the next entry
// that deploys, which is the order SyncPackages lays them out.
func (s *Store) MovePackage(game, id, key string, delta int) (Profile, error) {
	if delta != -1 && delta != 1 {
		return Profile{}, errors.New("a package moves one place at a time")
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		i := entryIndex(p.Entries, key)
		if i < 0 {
			return errors.New("no such mod in this profile")
		}
		for j := i + delta; j >= 0 && j < len(p.Entries); j += delta {
			if !p.Entries[j].IsOverlay() {
				p.Entries[i], p.Entries[j] = p.Entries[j], p.Entries[i]
				return nil
			}
		}
		return nil
	})
}

// MakePackageWin moves a mod to just after over in the package order, so it wins every file both provide.
func (s *Store) MakePackageWin(game, id, key, over string) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		i, j := entryIndex(p.Entries, key), entryIndex(p.Entries, over)
		if i < 0 || j < 0 {
			return errors.New("no such mod in this profile")
		}
		if i > j {
			return nil
		}
		moved := p.Entries[i]
		p.Entries = slices.Delete(p.Entries, i, i+1)
		p.Entries = slices.Insert(p.Entries, j, moved)
		return nil
	})
}
