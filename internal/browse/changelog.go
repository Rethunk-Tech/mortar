package browse

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Changelog reads the notes of the versions an update of project id on sourceID crosses, newest first. A source that
// keeps no changelog answers none.
func Changelog(ctx context.Context, game, sourceID, id, installed, latest, mortarVersion string) ([]nexus.Changelog, error) {
	info, ok := components.Game(game)
	if !ok {
		return nil, fmt.Errorf("unknown game %q", game)
	}
	gs, ok := info.Source(sourceID)
	if !ok {
		return nil, fmt.Errorf("game %q has no source %q", game, sourceID)
	}
	entry, ok := source.Get(sourceID)
	if !ok {
		return nil, fmt.Errorf("unknown source %q", sourceID)
	}
	c, ok := entry.Source.(source.Changelogger)
	if !ok {
		return []nexus.Changelog{}, nil
	}
	notes, err := c.Changelog(ctx, gs.Key, id, installed, latest, mortarVersion)
	if err != nil {
		return nil, err
	}
	out := make([]nexus.Changelog, 0, len(notes))
	for _, n := range notes {
		out = append(out, nexus.Changelog{Version: n.Version, Date: n.Date, Notes: []string{}, Body: n.Body})
	}
	return out, nil
}

// Changelog is the package-level Changelog at the service's Mortar version.
func (s *Service) Changelog(ctx context.Context, game, sourceID, id, installed, latest string) ([]nexus.Changelog, error) {
	return Changelog(ctx, game, sourceID, id, installed, latest, s.Version)
}
