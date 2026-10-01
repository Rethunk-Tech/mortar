package launchsvc

import (
	"fmt"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
)

const recentLaunchLimit = 3

// RecentLaunch is a profile Mortar recently launched, for the tray Play menu.
type RecentLaunch struct {
	ProfileID string    `json:"profileId"`
	Name      string    `json:"name"`
	When      time.Time `json:"-"`
}

type recentCandidate struct {
	id   string
	when time.Time
}

// RecentLaunches returns up to limit profiles most recently launched for game, newest first.
func (s *Service) RecentLaunches(gameID string, limit int) ([]RecentLaunch, error) {
	if game.Find(gameID) == nil {
		return nil, fmt.Errorf("unknown game %q", gameID)
	}
	if limit <= 0 {
		limit = recentLaunchLimit
	}
	names := map[string]string{}
	all, err := s.profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		names[p.ID] = p.Name
	}
	var candidates []recentCandidate
	if s.settings != nil {
		if played, ok := s.settings.Get().LastPlayed[gameID]; ok && played.Profile != "" && played.At != "" {
			if t, err := time.Parse(time.RFC3339, played.At); err == nil {
				candidates = append(candidates, recentCandidate{id: played.Profile, when: t})
			}
		}
	}
	for _, p := range all {
		modsDir, err := s.profiles.ModsDir(gameID, p.ID)
		if err != nil {
			continue
		}
		idx, err := readIndex(runsDir(modsDir))
		if err != nil || len(idx.Runs) == 0 {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, idx.Runs[0].Started)
		if err != nil {
			continue
		}
		candidates = append(candidates, recentCandidate{id: p.ID, when: t})
	}
	slices.SortFunc(candidates, func(a, b recentCandidate) int {
		return b.when.Compare(a.when)
	})
	seen := map[string]bool{}
	out := make([]RecentLaunch, 0, limit)
	for _, c := range candidates {
		if seen[c.id] {
			continue
		}
		seen[c.id] = true
		name := names[c.id]
		if name == "" {
			name = c.id
		}
		out = append(out, RecentLaunch{ProfileID: c.id, Name: name, When: c.when})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
