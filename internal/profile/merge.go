package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
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
		if manifest.LoaderManaged(side.ID) {
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
	changed := 0
	var done []string
	for _, side := range pv.Adds {
		if slices.Contains(done, side.Key) {
			continue
		}
		done = append(done, side.Key)
		if _, err := s.AddEntry(game, toID, side.Key, side.Source); err != nil {
			if _, dup := errors.AsType[*DuplicateError](err); dup {
				continue
			}
			return Profile{}, err
		}
		for _, m := range pv.Adds {
			if m.Key != side.Key {
				continue
			}
			changed++
			if !m.Enabled {
				if _, err := s.SetModEnabled(game, toID, m.Key, m.ID, false); err != nil {
					return Profile{}, err
				}
			}
			if err := s.carryConfig(game, fromID, toID, m); err != nil {
				return Profile{}, err
			}
		}
	}
	if newerWins {
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
				return Profile{}, err
			}
			for _, x := range pv.Both {
				if x.Target.Key == b.Target.Key && x.SourceNewer {
					changed++
				}
			}
		}
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

// carryConfig copies the source mod's config.json over the fresh copy in toID, when the source has one.
func (s *Store) carryConfig(game, fromID, toID string, side DiffSide) error {
	srcDir, err := s.ModFolder(game, fromID, side.Key, side.ID)
	if err != nil {
		return err
	}
	b, err := fsx.ReadFile(filepath.Join(srcDir, configFile))
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
	return datadir.WriteFile(filepath.Join(dstDir, configFile), b, 0o600)
}
