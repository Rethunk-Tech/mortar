package control

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

// updateChangelog returns the notes an update crosses: GitHub releases when repo is set, else the mod's Nexus changelog.
func (s *Services) updateChangelog(ctx context.Context, game string, modID int, repo, installed, latest string) ([]nexus.Changelog, error) {
	return s.Nexus.UpdateChangelog(ctx, game, modID, repo, installed, latest)
}
