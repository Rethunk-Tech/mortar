package profile

// PreviewEverywhere reports which profiles would receive the same store-key update.
func (s *Service) PreviewEverywhere(game, modKeyOrID string) (EverywherePreview, error) {
	return s.store.PreviewEverywhere(game, modKeyOrID)
}

// UpdateEverywhere replaces the matching entry in every eligible profile with newStoreKey
// (empty or "latest" means the newest store item that shares UniqueID / Nexus identity).
func (s *Service) UpdateEverywhere(game, modKeyOrID, newStoreKey string) (EverywhereResult, error) {
	return s.store.UpdateEverywhere(game, modKeyOrID, newStoreKey)
}
