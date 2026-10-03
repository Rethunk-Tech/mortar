package profile

// PreviewEverywhere reports which profiles would receive the same store-key update.
func (s *Service) PreviewEverywhere(game, modKeyOrUniqueID string) (EverywherePreview, error) {
	return s.store.PreviewEverywhere(game, modKeyOrUniqueID)
}

// UpdateEverywhere replaces the matching entry in every eligible profile with newStoreKey
// (empty or "latest" means the newest store item that shares UniqueID / Nexus identity).
func (s *Service) UpdateEverywhere(game, modKeyOrUniqueID, newStoreKey string) (EverywhereResult, error) {
	return s.store.UpdateEverywhere(game, modKeyOrUniqueID, newStoreKey)
}
