package datasvc

import (
	"cmp"
	"path/filepath"
	"sort"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// ProfileBudget is one profile's disk use split by where it lives. Store is what its mods occupy in the shared
// store, and Shared the part of that another profile of the game also uses, so it is not this profile's alone.
// Deployed is the profile's own mods folder plus the files its loader keeps in the profile, and Saves its separate
// saves (0 when the profile keeps none).
type ProfileBudget struct {
	Game     string `json:"game"`
	ID       string `json:"id"`
	Store    int64  `json:"store"`
	Shared   int64  `json:"shared"`
	Deployed int64  `json:"deployed"`
	Saves    int64  `json:"saves"`
}

// MeasureBudgets sizes every profile under root from the recorded store item sizes (game/key to bytes); only the
// profile's own folders are walked. savesOf reports whether a profile keeps its own saves folder, and loaderFiles names
// the paths, relative to the profile's folder, that its loader keeps there.
func MeasureBudgets(root string, itemSizes map[string]int64, savesOf func(game, id string) bool, loaderFiles func(game, id string) []string) ([]ProfileBudget, error) {
	dirs, err := datadir.ProfileDirs(filepath.Join(root, "profiles"))
	if err != nil {
		return nil, err
	}
	keysOf := make([][]string, len(dirs))
	users := map[string]int{}
	for i, d := range dirs {
		seen := map[string]bool{}
		for _, e := range readProfileEntries(filepath.Join(d.Dir, "profile.json")).Entries {
			for _, key := range append([]string{cmp.Or(e.Item, e.Key)}, e.ExtraStoreKeys...) {
				id := d.Game + "/" + key
				if key == "" || seen[id] {
					continue
				}
				seen[id] = true
				keysOf[i] = append(keysOf[i], id)
				users[id]++
			}
		}
	}
	out := make([]ProfileBudget, 0, len(dirs))
	for i, d := range dirs {
		b := ProfileBudget{Game: d.Game, ID: d.ID, Deployed: dirSize(filepath.Join(d.Dir, "mods"))}
		if loaderFiles != nil {
			for _, rel := range loaderFiles(d.Game, d.ID) {
				b.Deployed += dirSize(filepath.Join(d.Dir, filepath.FromSlash(rel)))
			}
		}
		for _, id := range keysOf[i] {
			b.Store += itemSizes[id]
			if users[id] > 1 {
				b.Shared += itemSizes[id]
			}
		}
		if savesOf != nil && savesOf(d.Game, d.ID) {
			b.Saves = dirSize(filepath.Join(d.Dir, "saves"))
		}
		out = append(out, b)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Game+"/"+out[a].ID < out[b].Game+"/"+out[b].ID })
	return out, nil
}

// budgetTTL bounds how stale a cached budget gets when only a profile's files changed, which the data folder's
// fingerprint does not see.
const budgetTTL = time.Minute

// ProfileBudgets is each profile's disk use split by store, shared, deployed and saves, cached until the store or
// the profiles change or a minute passes.
func (s *Service) ProfileBudgets() ([]ProfileBudget, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	fp := usageFingerprint(dir)
	s.mu.Lock()
	if s.budgetFP == fp && fp != "" && time.Since(s.budgetAt) < budgetTTL {
		out := s.budgets
		s.mu.Unlock()
		return out, nil
	}
	s.mu.Unlock()
	sizes, err := s.EntrySizes()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]int64, len(sizes))
	for _, e := range sizes {
		byKey[e.Game+"/"+e.Key] = e.Size
	}
	out, err := MeasureBudgets(dir, byKey, s.profiles.SeparateSaves, s.loaderFiles)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.budgets, s.budgetFP, s.budgetAt = out, fp, time.Now()
	s.mu.Unlock()
	return out, nil
}

// loaderFiles are the paths the profile's loader keeps in its folder, none for a loader that lives in the game's install.
func (s *Service) loaderFiles(gameID, id string) []string {
	if l, ok := game.LoaderOf(gameID, s.profiles.LoaderID(gameID, id)); ok {
		if in, ok := l.(loader.InProfile); ok {
			return in.ProfileFiles()
		}
	}
	return nil
}
