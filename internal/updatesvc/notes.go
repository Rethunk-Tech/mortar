package updatesvc

import (
	"context"
	"fmt"
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

// ReleaseNotes returns the notes of the Mortar release for version. A failed lookup is not an error: the result has
// Available false.
func (s *Service) ReleaseNotes(ctx context.Context, version string) ReleaseNotes {
	notes, err := s.releaseNotes(ctx, version)
	if err != nil || notes == "" {
		return ReleaseNotes{Version: version}
	}
	return ReleaseNotes{Version: version, Notes: notes, Available: true}
}
