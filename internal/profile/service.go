package profile

// Service exposes the store to the frontend.
type Service struct{ store *Store }

func NewService(store *Store) *Service { return &Service{store: store} }

func (s *Service) List(game string) ([]Profile, error) { return s.store.List(game) }

func (s *Service) Create(game, name string) (Profile, error) { return s.store.Create(game, name) }

func (s *Service) Rename(game, id, name string) (Profile, error) {
	return s.store.Rename(game, id, name)
}

// AddEntry copies the store item key into the profile.
func (s *Service) AddEntry(game, id, key string, source Source) (Profile, error) {
	return s.store.AddEntry(game, id, key, source)
}

func (s *Service) RemoveEntry(game, id, key string) (Profile, error) {
	return s.store.RemoveEntry(game, id, key)
}

func (s *Service) SetModEnabled(game, id, uniqueID string, enabled bool) (Profile, error) {
	return s.store.SetModEnabled(game, id, uniqueID, enabled)
}

func (s *Service) Duplicate(game, id string) (Profile, error) { return s.store.Duplicate(game, id) }

// Delete moves the profile to the trash, where it stays restorable for 30 days.
func (s *Service) Delete(game, id string) error { return s.store.Delete(game, id) }

func (s *Service) ListTrash(game string) ([]TrashItem, error) { return s.store.ListTrash(game) }

func (s *Service) Restore(game, id string) (Profile, error) { return s.store.Restore(game, id) }

func (s *Service) SetHidden(game, id string, hidden bool) (Profile, error) {
	return s.store.SetHidden(game, id, hidden)
}

func (s *Service) Reorder(game string, ids []string) error { return s.store.Reorder(game, ids) }

// Mods lists the profile's mods, rebuilding missing mod folders from the store first.
func (s *Service) Mods(game, id string) ([]Mod, error) { return s.store.Mods(game, id) }
