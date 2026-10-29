package profile

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// MergeBoth is a mod both profiles hold, with each profile's copy.
type MergeBoth struct {
	ID     mod.ID   `json:"id"`
	Name   string   `json:"name"`
	Source DiffSide `json:"source"`
	Target DiffSide `json:"target"`
	// SourceNewer is true when the source's version orders after the target's.
	SourceNewer bool `json:"sourceNewer"`
}

// MergePreview is what merging one profile into another would do.
type MergePreview struct {
	Adds       []DiffSide  `json:"adds"`
	Both       []MergeBoth `json:"both"`
	TargetOnly int         `json:"targetOnly"`
}

func indexMergeable(p Profile) map[string]DiffSide {
	out := indexUserMods(p)
	for k, side := range out {
		if smapimanifest.LoaderManaged(side.ID) {
			delete(out, k)
		}
	}
	return out
}

func (s *Store) MergePreview(game, fromID, toID string) (MergePreview, error) {
	if fromID == toID {
		return MergePreview{}, fmt.Errorf("pick a different profile to merge into")
	}
	from, err := s.load(game, fromID)
	if err != nil {
		return MergePreview{}, err
	}
	to, err := s.load(game, toID)
	if err != nil {
		return MergePreview{}, err
	}
	scheme := gamereg.VersionScheme(game)
	src, dst := indexMergeable(from), indexMergeable(to)
	out := MergePreview{Adds: []DiffSide{}, Both: []MergeBoth{}}
	for k, side := range src {
		other, ok := dst[k]
		if !ok {
			out.Adds = append(out.Adds, side)
			continue
		}
		c, ordered := deps.Compare(scheme, side.Version, other.Version)
		out.Both = append(out.Both, MergeBoth{ID: side.ID, Name: side.Name, Source: side, Target: other, SourceNewer: ordered && c > 0})
	}
	for k := range dst {
		if _, ok := src[k]; !ok {
			out.TargetOnly++
		}
	}
	sortSides(out.Adds)
	slices.SortFunc(out.Both, func(a, b MergeBoth) int { return compareNameThenID(a.Name, a.ID, b.Name, b.ID) })
	return out, nil
}

// MergeInto adds the mods only fromID holds to toID, with their enabled state and config, and with newerWins also
// switches each mod both hold to the source's version when that is newer. Nothing is removed from toID, fromID is
// untouched, and the whole merge is one history event on toID.
func (s *Store) MergeInto(game, fromID, toID string, newerWins bool) (Profile, error) {
	pv, err := s.MergePreview(game, fromID, toID)
	if err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	err = s.unlocked(game, toID)
	s.mu.Unlock()
	if err != nil {
		return Profile{}, err
	}
	from, err := s.load(game, fromID)
	if err != nil {
		return Profile{}, err
	}
	s.setHistoryQuiet(toID, true)
	defer s.setHistoryQuiet(toID, false)
	// A failure takes back the entries the merge added; a version it already switched stays switched.
	added, changed, err := s.mergeAdds(game, fromID, toID, pv, from)
	if err == nil && newerWins {
		var swapped int
		swapped, err = s.mergeSwaps(game, toID, pv)
		changed += swapped
	}
	if err != nil {
		if len(added) > 0 {
			if _, undo := s.RemoveEntries(game, toID, added); undo != nil {
				log.Printf("profile: taking back a failed merge into %s: %v", toID, undo)
			}
		}
		return Profile{}, err
	}
	if changed == 0 {
		return s.load(game, toID)
	}
	change, err := s.recordSnapshot(game, toID, historyBulk, HistoryEvent{Change: ChangeMerged, Name: from.Name}, changed)
	if err != nil {
		return Profile{}, err
	}
	if err := s.RecordModsSnapshot(game, toID); err != nil {
		return Profile{}, err
	}
	p, err := s.load(game, toID)
	p.LastChange = change
	return p, err
}

// mergeAdds adds each mod only fromID holds to toID with its enabled state and config, and returns the entries it
// added and how many mods they hold.
func (s *Store) mergeAdds(game, fromID, toID string, pv MergePreview, from Profile) ([]string, int, error) {
	changed := 0
	var done, added []string
	for _, side := range pv.Adds {
		if slices.Contains(done, side.Key) {
			continue
		}
		done = append(done, side.Key)
		src := side.Source
		if i := entryIndex(from.Entries, side.Key); i >= 0 {
			src = src.WithImplicit(from.Entries[i].Implicit)
		}
		if _, err := s.AddEntry(game, toID, side.Key, src); err != nil {
			if _, dup := errors.AsType[*DuplicateError](err); dup {
				continue
			}
			return added, changed, err
		}
		added = append(added, side.Key)
		for _, m := range pv.Adds {
			if m.Key != side.Key {
				continue
			}
			changed++
			if !m.Enabled {
				if _, err := s.SetModEnabled(game, toID, m.Key, m.ID, false); err != nil {
					return added, changed, err
				}
			}
			if err := s.carryConfig(game, fromID, toID, m); err != nil {
				return added, changed, err
			}
		}
	}
	return added, changed, nil
}

// mergeSwaps switches each mod both profiles hold to the source's version where that is newer, and returns how many
// mods changed.
func (s *Store) mergeSwaps(game, toID string, pv MergePreview) (int, error) {
	changed := 0
	var swapped []string
	for _, b := range pv.Both {
		if !b.SourceNewer || slices.Contains(swapped, b.Target.Key) {
			continue
		}
		swapped = append(swapped, b.Target.Key)
		src := b.Source.Source
		if _, err := s.moveTo(game, toID, b.Target.Key, b.Source.Key, &src); err != nil {
			if _, dup := errors.AsType[*DuplicateError](err); dup {
				continue
			}
			return changed, err
		}
		for _, x := range pv.Both {
			if x.Target.Key == b.Target.Key && x.SourceNewer {
				changed++
			}
		}
	}
	return changed, nil
}

// carryConfig copies the source mod's config.json over the fresh copy in toID, when the source has one.
func (s *Store) carryConfig(game, fromID, toID string, side DiffSide) error {
	srcDir, err := s.ModFolder(game, fromID, side.Key, side.ID)
	// A package keeps its settings in the loader's config folder, not a folder of its own: there is nothing to carry.
	if errors.Is(err, ErrNoModFolder) {
		return nil
	}
	if err != nil {
		return err
	}
	b, err := safefs.ReadFile(filepath.Join(srcDir, configFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	dstDir, err := s.ModFolder(game, toID, side.Key, side.ID)
	if err != nil {
		return err
	}
	return safefs.WriteFileAtomic(filepath.Join(dstDir, configFile), b, 0o600)
}
