package updatesvc

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

// ReleaseNotes is one release's notes as the What's New dialog shows them. Available is false when the notes
// could not be fetched (offline, rate limited, no such release), so the window shows "notes unavailable".
type ReleaseNotes struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Available bool   `json:"available"`
}

func (s *Service) ghClient() *github.Client {
	if s.gh != nil {
		return s.gh
	}
	return &github.Client{CacheDir: filepath.Join(s.dir, "cache")}
}

func (s *Service) releaseNotes(ctx context.Context, version string) (string, error) {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if v == "" {
		return "", fmt.Errorf("empty version")
	}
	owner, repo, _ := strings.Cut(mortarRepo, "/")
	return s.ghClient().ReleaseNotes(ctx, owner, repo, "v"+v)
}

// skippedReleases is the notes of the releases newer than last and older than cur, newest first; none when GitHub
// cannot be read, since the current release's notes still show.
func (s *Service) skippedReleases(ctx context.Context, last, cur string) []ReleaseNotes {
	owner, repo, _ := strings.Cut(mortarRepo, "/")
	rels, err := s.ghClient().ReleasesBetween(ctx, owner, repo, last, cur)
	if err != nil {
		slog.Warn("what's new", "last", last, "err", err)
		return nil
	}
	var out []ReleaseNotes
	for _, r := range rels {
		v := strings.TrimPrefix(r.Tag, "v")
		if v == cur || strings.TrimSpace(r.Body) == "" {
			continue
		}
		out = append(out, ReleaseNotes{Version: v, Notes: strings.TrimSpace(r.Body), Available: true})
	}
	return out
}

// ReleaseNotes returns the notes of the Mortar release for version. A failed lookup is not an error: the result has
// Available false.
func (s *Service) ReleaseNotes(ctx context.Context, version string) ReleaseNotes {
	notes, err := s.releaseNotes(ctx, version)
	if err != nil || notes == "" {
		return ReleaseNotes{Version: version}
	}
	return ReleaseNotes{Version: version, Notes: notes, Available: true}
}
