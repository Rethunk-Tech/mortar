package profile

import "fmt"

const historyCollectionUnlink = "collection-unlink"

// ClearCollection removes the Nexus collection link from a profile without changing its mods.
func (s *Store) ClearCollection(game, id string) (Profile, error) {
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	if p.Collection == nil {
		return p, nil
	}
	name := p.Collection.Name
	label := "Unlinked collection"
	if name != "" {
		label = fmt.Sprintf("Unlinked collection %s", name)
	}
	return s.updateLockedAs(game, id, historyCollectionUnlink, label, func(p *Profile, _ string) error {
		p.Collection = nil
		return nil
	})
}
