package profile

import "github.com/Rethunk-Tech/mortar/internal/mod"

// ConfigView answers the config questions about many mods of one profile from a single read of its profile.json,
// where each Store method reads (and deep-copies) the whole profile again. Each func matches the Store method of the
// same name, less the game and profile.
type ConfigView struct {
	ModFolder     func(key string, uniqueID mod.ID) (string, error)
	ReadConfig    func(key string, uniqueID mod.ID) (string, error)
	ShippedConfig func(key string, uniqueID mod.ID) (string, bool)
	PluginGUIDs   func(uniqueID mod.ID) []string
}

// ConfigView reads the profile once.
func (s *Store) ConfigView(game, id string) (ConfigView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return ConfigView{}, err
	}
	return ConfigView{
		ModFolder: func(key string, uniqueID mod.ID) (string, error) { return modFolderIn(p, dir, key, uniqueID) },
		ReadConfig: func(key string, uniqueID mod.ID) (string, error) {
			folder, err := modFolderIn(p, dir, key, uniqueID)
			if err != nil {
				return "", err
			}
			path, err := configPathIn(folder)
			if err != nil {
				return "", err
			}
			return readConfigAt(path)
		},
		ShippedConfig: func(key string, uniqueID mod.ID) (string, bool) { return s.shippedConfigOf(game, p, key, uniqueID) },
		PluginGUIDs:   func(uniqueID mod.ID) []string { return s.pluginGUIDsOf(game, p, uniqueID) },
	}, nil
}
