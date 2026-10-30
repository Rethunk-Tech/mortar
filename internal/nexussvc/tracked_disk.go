package nexussvc

import (
	"context"
	"slices"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	trackedFile = "nexus-tracked.json"
	// TrackedEvent is emitted when a background refresh found a different tracked list.
	TrackedEvent = "nexus:tracked"
)

// trackedDisk is the tracked list as last fetched; User ties it to the account that fetched it.
type trackedDisk struct {
	User int                    `json:"user"`
	Mods []nexusmods.TrackedMod `json:"mods"`
}

func (s *Service) readTracked() ([]nexusmods.TrackedMod, bool) {
	var d trackedDisk
	found, err := datadir.ReadJSON(s.trackedPath, &d)
	if s.trackedPath == "" || err != nil || !found || d.User == 0 || d.User != s.store.Get().NexusUserID {
		return nil, false
	}
	return d.Mods, true
}

// forgetTracked drops the kept list: after a sign-out, and after a track or untrack that made it stale.
func (s *Service) forgetTracked() {
	if s.trackedPath != "" {
		_ = safefs.Remove(s.trackedPath)
	}
}

// fetchTracked reads the list from Nexus and keeps it, unless the account changed meanwhile.
func (s *Service) fetchTracked(ctx context.Context) ([]nexusmods.TrackedMod, error) {
	user := s.store.Get().NexusUserID
	mods, err := s.TrackedMods(ctx)
	if err != nil {
		return nil, err
	}
	if s.trackedPath != "" && user != 0 && user == s.store.Get().NexusUserID {
		_ = datadir.WriteJSON(s.trackedPath, trackedDisk{User: user, Mods: mods})
	}
	return mods, nil
}

// trackedForCard answers from the kept list at once and refreshes it in the background; with no kept list it waits
// for Nexus.
func (s *Service) trackedForCard(ctx context.Context) ([]nexusmods.TrackedMod, error) {
	kept, ok := s.readTracked()
	if !ok {
		return s.fetchTracked(ctx)
	}
	if s.refreshing.CompareAndSwap(false, true) {
		go func() {
			defer s.refreshing.Store(false)
			bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
			defer cancel()
			fresh, err := s.fetchTracked(bg)
			if err == nil && !slices.Equal(fresh, kept) && s.App != nil {
				s.App.Emit(TrackedEvent, "")
			}
		}()
	}
	return kept, nil
}
