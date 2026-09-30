package profile

// UpdateEntry replaces the entry oldKey with the store item newKey, carrying over the mod's and the user's files.
func (s *Service) UpdateEntry(game, id, oldKey, newKey string) (Profile, error) {
	return s.store.UpdateEntry(game, id, oldKey, newKey)
}

// RollBack switches the entry back to its previous version.
func (s *Service) RollBack(game, id, key string) (Profile, error) {
	return s.store.RollBack(game, id, key)
}
