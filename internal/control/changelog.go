package control

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

func (s *Services) changelogBetween(ctx context.Context, game string, modID int, installed, latest string) ([]nexus.Changelog, error) {
	if modID < 1 {
		return []nexus.Changelog{}, nil
	}
	return s.Nexus.ChangelogBetween(ctx, game, modID, installed, latest)
}
