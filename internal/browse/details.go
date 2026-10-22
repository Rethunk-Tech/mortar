package browse

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

type detailsEntry struct {
	details source.Details
	until   time.Time
}

// details remembers each mod's Details for topTTL, so reopening a card asks no site again but one left open in the
// tray for days does not show old versions. It is keyed by game, source and id; Mortar runs one browse service.
var details = struct {
	sync.Mutex
	m map[string]detailsEntry
}{m: map[string]detailsEntry{}}

// Details reads the fuller page of one hit: its README, versions, dependencies and categories. A source that keeps no
// such page answers empty Details; a Nexus mod's page comes from the Nexus details the window already loads.
func (s *Service) Details(ctx context.Context, game, sourceID, id string) (source.Details, error) {
	key := game + "\x00" + sourceID + "\x00" + id
	details.Lock()
	hit, ok := details.m[key]
	details.Unlock()
	if ok && time.Now().Before(hit.until) {
		return hit.details, nil
	}
	info, ok := catalogGame(game)
	if !ok {
		return source.Details{}, fmt.Errorf("unknown game %q", game)
	}
	gs, ok := info.Source(sourceID)
	if !ok {
		return source.Details{}, fmt.Errorf("game %q has no source %q", game, sourceID)
	}
	entry, ok := source.Get(sourceID)
	if !ok {
		return source.Details{}, fmt.Errorf("unknown source %q", sourceID)
	}
	d, ok := entry.Source.(source.Detailer)
	if !ok {
		return source.Details{Versions: []string{}, Dependencies: []string{}, Categories: []string{}}, nil
	}
	got, err := d.Details(ctx, gs.Key, id, s.Version)
	if err != nil {
		return source.Details{}, err
	}
	now := time.Now()
	details.Lock()
	maps.DeleteFunc(details.m, func(_ string, e detailsEntry) bool { return !now.Before(e.until) })
	details.m[key] = detailsEntry{details: got, until: now.Add(topTTL)}
	details.Unlock()
	return got, nil
}
