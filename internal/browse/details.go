package browse

import (
	"context"
	"fmt"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// details remembers each mod's Details for the session, so reopening a card asks no site again. It is keyed by
// game, source and id; Mortar runs one browse service.
var details = struct {
	sync.Mutex
	m map[string]source.Details
}{m: map[string]source.Details{}}

// Details reads the fuller page of one hit: its README, versions, dependencies and categories. A source that keeps no
// such page answers empty Details; a Nexus mod's page comes from the Nexus details the window already loads.
func (s *Service) Details(ctx context.Context, game, sourceID, id string) (source.Details, error) {
	key := game + "\x00" + sourceID + "\x00" + id
	details.Lock()
	hit, ok := details.m[key]
	details.Unlock()
	if ok {
		return hit, nil
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
	details.Lock()
	details.m[key] = got
	details.Unlock()
	return got, nil
}
