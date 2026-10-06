package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Details reads a repository's README. Its releases, which are its versions and changelog, come from the releases
// list the update check already caches.
func (d *Driver) Details(ctx context.Context, _, id, mortarVersion string) (source.Details, error) {
	owner, repo, ok := strings.Cut(id, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return source.Details{}, fmt.Errorf("%q is not an owner/repo", id)
	}
	base := d.URL
	if base == "" {
		base = apiURL
	}
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	u := strings.TrimRight(base, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/readme"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return source.Details{}, err
	}
	req.Header.Set("User-Agent", source.UserAgent(mortarVersion))
	req.Header.Set("Accept", "application/vnd.github.raw")
	resp, err := client.Do(req)
	if err != nil {
		return source.Details{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	out := source.Details{Versions: []string{}, Dependencies: []string{}, Categories: []string{}}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return out, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		return source.Details{}, source.Busy("GitHub", resp)
	default:
		return source.Details{}, fmt.Errorf("github answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return source.Details{}, err
	}
	out.Description = string(body)
	return out, nil
}
