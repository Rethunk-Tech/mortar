// Package gamestore finds a game's installs through one driver per store the component catalog can name.
package gamestore

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/gog"
	"github.com/Rethunk-Tech/mortar/internal/lutris"
)

// Install store ids, as saved in settings and shown to the frontend.
const (
	StoreSteam        = "steam"
	StoreFlatpakSteam = "flatpak-steam"
	StoreGOG          = gog.StoreGOG
	StoreGOGHeroic    = gog.StoreHeroic
	StoreMinigalaxy   = gog.StoreMinigalaxy
	StoreLutris       = lutris.StoreLutris
	StoreBottles      = "bottles"
	StoreEA           = "ea"
)

// Launcher ids, each the source of one or more stores' installs and the key of the user's added folders.
const (
	LauncherSteam        = "steam"
	LauncherFlatpakSteam = "flatpak-steam"
	LauncherHeroic       = "heroic"
	LauncherLutris       = "lutris"
	LauncherGOG          = "gog"
	LauncherMinigalaxy   = "minigalaxy"
	LauncherBottles      = "bottles"
	LauncherEA           = "ea"
)

// Install is one discovered game folder and the store it came from.
type Install struct {
	Store, Dir string
	// Prefix is the Wine prefix a Windows build runs in; empty unless the store keeps one (Bottles).
	Prefix string
}

// LauncherSpec describes one launcher (a store app) a driver reads.
type LauncherSpec struct {
	ID, Name string
	// Looked lists the folders searched for this launcher, the user's own first; added holds every launcher's added
	// folders, since one launcher's folder can rule out another's.
	Looked func(home string, added map[string][]string) []string
	// Usable reports whether a folder is this launcher's.
	Usable func(dir string) bool
}

// Store finds installs of games the catalog lists under the store's key.
type Store interface {
	// Key is the catalog's `stores` key for this store.
	Key() string
	Launchers(goos string) []LauncherSpec
	// Discover returns g's installs; roots are the user's added folders by launcher id.
	Discover(home string, roots map[string][]string, g components.GameInfo) []Install
}

// All returns the drivers in discovery order.
func All() []Store {
	return []Store{steamStore{}, gogStore{}, lutrisStore{}, bottlesStore{}, eaStore{}}
}

// Has reports whether g's catalog entry names the store key.
func Has(g components.GameInfo, key string) bool {
	switch key {
	case steamKey:
		return g.Stores.Steam != nil
	case gogKey:
		return g.Stores.GOG != nil
	case lutrisKey:
		return g.Stores.Lutris != nil
	case bottlesKey:
		return g.Stores.Bottles != nil
	case eaKey:
		return g.Stores.EA != nil
	}
	return false
}

// Discover returns g's installs from every driver its catalog entry names, de-duplicated by folder.
func Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	var all []Install
	seen := map[string]struct{}{}
	for _, st := range All() {
		if !Has(g, st.Key()) {
			continue
		}
		for _, in := range st.Discover(home, roots, g) {
			if _, ok := seen[in.Dir]; ok || in.Dir == "" {
				continue
			}
			seen[in.Dir] = struct{}{}
			all = append(all, in)
		}
	}
	return all
}

var storeOrder = []string{StoreSteam, StoreFlatpakSteam, StoreGOG, StoreGOGHeroic, StoreMinigalaxy, StoreEA, StoreLutris, StoreBottles}

// Rank orders stores for choosing a default install, lowest first; an unknown store ties with Steam.
func Rank(store string) int { return max(slices.Index(storeOrder, store), 0) }

// launcherOrder is the order the setup screen lists launchers in.
var launcherOrder = []string{LauncherSteam, LauncherFlatpakSteam, LauncherHeroic, LauncherLutris, LauncherMinigalaxy, LauncherBottles, LauncherGOG, LauncherEA}

// Launchers returns every driver's launchers on goos in the setup screen's order.
func Launchers(goos string) []LauncherSpec {
	var out []LauncherSpec
	for _, st := range All() {
		out = append(out, st.Launchers(goos)...)
	}
	slices.SortStableFunc(out, func(a, b LauncherSpec) int {
		return slices.Index(launcherOrder, a.ID) - slices.Index(launcherOrder, b.ID)
	})
	return out
}

// LauncherOf is the launcher that reported installs of store.
func LauncherOf(store string) string {
	switch store {
	case StoreGOGHeroic:
		return LauncherHeroic
	case StoreMinigalaxy:
		return LauncherMinigalaxy
	}
	return store
}
