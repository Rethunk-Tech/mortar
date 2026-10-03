package profile

func (s *Service) CreateGroup(game, id, name string) (Profile, error) {
	return s.store.CreateGroup(game, id, name)
}

func (s *Service) RenameGroup(game, id, name, next string) (Profile, error) {
	return s.store.RenameGroup(game, id, name, next)
}

func (s *Service) DeleteGroup(game, id, name string) (Profile, error) {
	return s.store.DeleteGroup(game, id, name)
}

func (s *Service) AddToGroup(game, id, name, key string) (Profile, error) {
	return s.store.AddToGroup(game, id, name, key)
}

func (s *Service) RemoveFromGroup(game, id, name, key string) (Profile, error) {
	return s.store.RemoveFromGroup(game, id, name, key)
}

func (s *Service) SetGroupEnabled(game, id, name string, on bool) (Profile, error) {
	return s.store.SetGroupEnabled(game, id, name, on)
}
