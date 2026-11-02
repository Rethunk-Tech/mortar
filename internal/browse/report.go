package browse

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// ReportTarget says where to report a problem with the mod id on sourceID: its GitHub repository when the site links
// one, else a page. A source that cannot say answers an empty target.
func (s *Service) ReportTarget(ctx context.Context, game, sourceID, id string) (source.ReportTarget, error) {
	info, ok := components.Game(game)
	if !ok {
		return source.ReportTarget{}, fmt.Errorf("unknown game %q", game)
	}
	gs, ok := info.Source(sourceID)
	if !ok {
		return source.ReportTarget{}, fmt.Errorf("game %q has no source %q", game, sourceID)
	}
	entry, ok := source.Get(sourceID)
	if !ok {
		return source.ReportTarget{}, fmt.Errorf("unknown source %q", sourceID)
	}
	r, ok := entry.Source.(source.Reporter)
	if !ok {
		return source.ReportTarget{}, nil
	}
	return r.ReportTarget(ctx, gs.Key, id, s.Version)
}
