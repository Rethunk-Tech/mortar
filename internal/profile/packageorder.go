package profile

import "errors"

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
