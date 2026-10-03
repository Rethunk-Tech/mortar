package nexussvc

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
)

const changelogTTL = 4 * time.Hour

// ChangelogBetween returns Nexus changelog entries strictly newer than installed, up to latest, newest first.
func (s *Service) ChangelogBetween(ctx context.Context, game string, modID int, installed, latest string) ([]nexus.Changelog, error) {
	all, err := s.changelogs(ctx, game, modID)
	if err != nil {
		return nil, err
	}
	return nexus.ChangelogBetween(all, installed, latest), nil
}

func (s *Service) changelogs(ctx context.Context, game string, modID int) ([]nexus.Changelog, error) {
	domain := nexus.Game
	if info, ok := components.BundledGame(game); ok && info.Nexus.Domain != "" {
		domain = info.Nexus.Domain
	}
	return meta.Cached(s.meta, fmt.Sprintf("nexus/changelogs-%s-%d.json", domain, modID), changelogTTL, func() ([]nexus.Changelog, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return nil, err
		}
		return c.Changelogs(ctx, modID, math.MaxInt)
	})
}
