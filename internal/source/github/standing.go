package github

import (
	"context"
	"errors"
	"net/http"
	"strings"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// standingLimit caps the repositories asked after in one pass, since GitHub has no batch read and its anonymous
// limit is sixty calls an hour.
const standingLimit = 20

// Standings flags the repositories ("owner/repo") that GitHub marks archived. It asks one repository at a time, at
// most standingLimit of them, and stops with GitHub's busy error when the limit is reached.
func (d *Driver) Standings(ctx context.Context, version string, ids []string) (map[string]source.Standing, error) {
	out := map[string]source.Standing{}
	for _, id := range ids[:min(len(ids), standingLimit)] {
		archived, err := d.archived(ctx, version, id)
		if err != nil {
			return nil, err
		}
		if archived {
			out[id] = source.Standing{State: "archived"}
		}
	}
	return out, nil
}

func (d *Driver) archived(ctx context.Context, _, repo string) (bool, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return false, nil
	}
	archived, err := (&gh.Client{HTTP: d.HTTP, APIBase: d.URL}).Archived(ctx, owner, name)
	if errors.Is(err, httpkit.ErrNotFound) || errors.Is(err, gh.ErrInvalidRepo) {
		return false, nil
	}
	if err != nil {
		return false, githubErr(err)
	}
	return archived, nil
}

func githubErr(err error) error {
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		if se.Code == http.StatusForbidden || se.Code == http.StatusTooManyRequests {
			return &source.BusyError{Source: "GitHub"}
		}
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &source.BusyError{Source: "GitHub", Reset: rl.Reset}
	}
	return answered("github", err)
}
