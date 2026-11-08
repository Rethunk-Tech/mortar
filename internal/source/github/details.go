package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Details reads a repository's README. Its releases, which are its versions and changelog, come from the releases
// list the update check already caches.
func (d *Driver) Details(ctx context.Context, _, id, _ string) (source.Details, error) {
	owner, repo, ok := strings.Cut(id, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return source.Details{}, fmt.Errorf("%q is not an owner/repo", id)
	}
	out := source.Details{Versions: []string{}, Dependencies: []string{}, Categories: []string{}}
	text, err := (&gh.Client{HTTP: d.HTTP, APIBase: d.URL}).Readme(ctx, owner, repo)
	if errors.Is(err, httpkit.ErrNotFound) || errors.Is(err, gh.ErrInvalidRepo) {
		return out, nil
	}
	if err != nil {
		return source.Details{}, githubErr(err)
	}
	out.Description = text
	return out, nil
}
