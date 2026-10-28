package settings

import (
	"slices"
	"strings"
)

// FavouriteKey is how a starred mod is named: its source and the id the source gives it, in lower case. The window
// builds the same key for a Browse hit and for an installed mod of that source.
func FavouriteKey(source, id string) string {
	return strings.ToLower(source + ":" + id)
}

// localSep joins a local archive's file name to its store key in its star key.
const localSep = "|"

// LocalFavouriteKey names a starred local archive by its store key, so two archives with one file name keep their own
// stars. The file name leads it because 0.3.8 stored "local:<file name>"; see legacyLocalKey.
func LocalFavouriteKey(name, storeKey string) string {
	return FavouriteKey("local", name+localSep+storeKey)
}

// legacyLocalKey is the 0.3.8 star key of the archive a LocalFavouriteKey names, or "" for any other key.
func legacyLocalKey(key string) string {
	head, _, ok := strings.Cut(key, localSep)
	if !ok || !strings.HasPrefix(head, "local:") {
		return ""
	}
	return head
}

// SetModFavourite stars a mod for the game, or removes its star.
func (s *Service) SetModFavourite(game, key string, favourite bool) error {
	key = strings.ToLower(strings.TrimSpace(key))
	legacy := legacyLocalKey(key)
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		kept := slices.DeleteFunc(slices.Clone(gp.FavouriteMods), func(k string) bool { return k == key || (legacy != "" && k == legacy) })
		if favourite && key != "" {
			kept = append(kept, key)
		}
		gp.FavouriteMods = kept
		putGame(v, game, gp)
	})
}
