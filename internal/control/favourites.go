package control

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// entryFavouriteKey is the star key of the mod an entry came from, as the window builds it; empty for a mod with no
// site.
func entryFavouriteKey(src profile.Source) string {
	id := src.Name
	switch src.Kind {
	case "nexus":
		if src.ModID == 0 {
			return ""
		}
		id = strconv.Itoa(src.ModID)
	case "github":
		id = src.Repo
	}
	if id == "" {
		return ""
	}
	return settings.FavouriteKey(src.Kind, id)
}

// setModFavourites stars or unstars the profile's mods for their game and returns the game's starred keys.
func (s *Services) setModFavourites(game string, prof profile.Profile, ids []string, on bool) ([]string, error) {
	if s.SettingsSvc == nil {
		return nil, errors.New("settings are unavailable")
	}
	keys, err := keysFor(prof, ids)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		var star string
		for _, e := range prof.Entries {
			if e.Key == k {
				star = entryFavouriteKey(e.Source)
			}
		}
		if star == "" {
			return nil, fmt.Errorf("%s came from no mod site, so it cannot be starred", k)
		}
		if err := s.SettingsSvc.SetModFavourite(game, star, on); err != nil {
			return nil, err
		}
	}
	return s.Settings.Get().GamePrefs(game).FavouriteMods, nil
}
