package profile

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
	note := HistoryEvent{Change: ChangeCollectionUnlinked, Name: p.Collection.Name}
	return s.updateLockedAs(game, id, historyCollectionUnlink, note, func(p *Profile, _ string) error {
		p.Collection = nil
		return nil
	})
}
