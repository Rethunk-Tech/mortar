package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

// pinTemplates records the commit each mod template's tag points at, so the signed catalog names the exact files
// Mortar downloads. A commit already in the source must still be the tag's: a tag that moved since stops the build.
func pinTemplates(ctx context.Context, client *http.Client, apiBase string, games []components.GameInfo) error {
	for _, g := range games {
		for _, l := range g.Loaders {
			t := l.DevTemplate
			if t == nil {
				continue
			}
			sha, err := tagCommit(ctx, client, apiBase, t.Repo, t.Ref)
			if err != nil {
				return fmt.Errorf("game %q mod template: %w", g.ID, err)
			}
			if t.Commit != "" && t.Commit != sha {
				return fmt.Errorf("game %q mod template %s@%s is now commit %s, not the recorded %s", g.ID, t.Repo, t.Ref, sha, t.Commit)
			}
			t.Commit = sha
		}
	}
	return nil
}

func tagCommit(ctx context.Context, client *http.Client, apiBase, repo, ref string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/repos/"+repo+"/commits/"+ref, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.sha")
	body, err := httpkit.DoCapped(client, req, 128, nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
