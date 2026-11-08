package profile

import (
	"maps"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-dotnet"
)

// PluginTwins maps the key of each package not installed from Thunderstore to the Thunderstore package
// ("Namespace-Name") that ships a plugin with the same BepInPlugin GUID, read from the game's installed packages in
// every profile, so no request is made. plugins lists the plugins a package folder declares.
func (s *Store) PluginTwins(gameID string, plugins func(dir string) []dotnet.Plugin) map[string]string {
	profiles, err := s.List(gameID)
	if err != nil {
		return nil
	}
	byGUID := map[string]string{}
	var others []PackageRef
	for _, p := range profiles {
		pkgs, _ := s.EnabledPackages(gameID, p.ID)
		for _, pk := range pkgs {
			if pk.Source != KindThunderstore {
				others = append(others, pk)
				continue
			}
			for _, pl := range plugins(pk.Dir) {
				byGUID[strings.ToLower(pl.GUID)] = pk.Name
			}
		}
	}
	twins := map[string]string{}
	for _, pk := range others {
		for _, pl := range plugins(pk.Dir) {
			if name, ok := byGUID[strings.ToLower(pl.GUID)]; ok && pl.GUID != "" {
				twins[pk.Key] = name
				break
			}
		}
	}
	return twins
}

// SetSources gives each entry of found that still has no source its page on a site, recording the installed version.
// An entry that already has a source keeps it.
func (s *Store) SetSources(game, id string, found map[string]Source) (Profile, error) {
	note := HistoryEvent{Change: ChangeSourced}
	return s.updateEntriesAs(game, id, historyBulk, note, slices.Sorted(maps.Keys(found)), func(e *Entry, key string) error {
		if !e.Source.Unsourced() {
			return nil
		}
		src := found[key]
		if len(e.Mods) > 0 && src.Version == "" {
			src.Version = e.Mods[0].Version
		}
		e.Source = src
		return nil
	})
}
