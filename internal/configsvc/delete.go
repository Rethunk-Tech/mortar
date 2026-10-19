package configsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DeleteUnowned deletes a config file that no mod of the profile owns (one the Config page lists under Loader and
// other). A file a mod owns is refused, and so is any change while the profile's game runs.
func (s *Service) DeleteUnowned(game, profileID, file string) error {
	if err := s.refuseRunning(game, profileID); err != nil {
		return err
	}
	list, err := s.Mods(game, profileID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(list.Other, func(f ModFile) bool { return f.Name == file })
	if i < 0 {
		return fmt.Errorf("%q is not a config file without an owner in this profile", file)
	}
	if strings.HasSuffix(strings.ToLower(file), cfgExt) {
		if guid := s.cfgGUID(game, profileID, file); s.installedPlugin(game, profileID, strings.TrimSuffix(file, filepath.Ext(file)), guid) {
			return fmt.Errorf("%q belongs to a plugin installed in this profile", file)
		}
	}
	var path string
	switch list.Other[i].Format {
	case FormatJSON:
		path, err = s.jsonPath(game, profileID, file)
	case FormatText:
		path, err = s.textPath(game, profileID, file)
	default:
		path, err = s.cfgPath(game, profileID, file)
	}
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// cfgGUID is the plugin GUID a .cfg's header names, "" when it has none.
func (s *Service) cfgGUID(game, profileID, file string) string {
	docs, err := s.cfgDocs(game, profileID)
	if err != nil {
		return ""
	}
	if i := slices.IndexFunc(docs, func(d cfgOnDisk) bool { return d.name == file }); i >= 0 {
		return docs[i].doc.guid
	}
	return ""
}

// installedPlugin reports whether the file's name or header GUID is the GUID of a plugin the profile holds.
func (s *Service) installedPlugin(game, profileID string, names ...string) bool {
	return slices.ContainsFunc(s.Profiles.InstalledPluginGUIDs(game, profileID), func(g string) bool {
		return slices.ContainsFunc(names, func(n string) bool { return n != "" && strings.EqualFold(n, g) })
	})
}
