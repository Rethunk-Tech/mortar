package nexussvc

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
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

// UpdateChangelog returns the notes of the versions an update crosses, newest first: GitHub releases (at most
// github.MaxChangelog) when the update comes from githubRepo ("owner/repo"), else the mod's Nexus changelog. Both
// ride the clients' own caches.
func (s *Service) UpdateChangelog(ctx context.Context, game string, modID int, githubRepo, installed, latest string) ([]nexus.Changelog, error) {
	if githubRepo == "" {
		if modID < 1 {
			return []nexus.Changelog{}, nil
		}
		return s.ChangelogBetween(ctx, game, modID, installed, latest)
	}
	owner, repo, _ := strings.Cut(githubRepo, "/")
	rels, err := s.githubClient().ReleasesBetween(ctx, owner, repo, installed, latest)
	if err != nil {
		return nil, err
	}
	return releaseChangelogs(rels), nil
}

// ReleaseChangelog returns every published release of githubRepo ("owner/repo") newest first, from the cached
// releases list the update check already reads.
func (s *Service) ReleaseChangelog(ctx context.Context, githubRepo string) ([]nexus.Changelog, error) {
	owner, repo, _ := strings.Cut(githubRepo, "/")
	rels, err := s.githubClient().Releases(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return releaseChangelogs(rels), nil
}

func (s *Service) githubClient() *github.Client {
	if s.GitHub == nil {
		return &github.Client{}
	}
	return s.GitHub
}

func releaseChangelogs(rels []github.Release) []nexus.Changelog {
	out := make([]nexus.Changelog, 0, len(rels))
	for _, r := range rels {
		if r.Draft {
			continue
		}
		date, _, _ := strings.Cut(r.Published, "T")
		out = append(out, nexus.Changelog{Version: strings.TrimLeft(r.Tag, "vV"), Date: date, Notes: []string{}, Body: strings.TrimSpace(r.Body)})
	}
	return out
}

func (s *Service) changelogs(ctx context.Context, gameID string, modID int) ([]nexus.Changelog, error) {
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return nil, err
	}
	return meta.Cached(s.meta, fmt.Sprintf("%s%s-%d.json", meta.NexusChangelogsPrefix, t.Domain, modID), changelogTTL, func() ([]nexus.Changelog, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return nil, err
		}
		return c.Changelogs(ctx, t, modID, math.MaxInt)
	})
}
