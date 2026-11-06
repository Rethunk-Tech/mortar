package github

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Details reads a repository's README. Its releases, which are its versions and changelog, come from the releases
// list the update check already caches.
func (d *Driver) Details(ctx context.Context, _, id, mortarVersion string) (source.Details, error) {
	owner, repo, ok := strings.Cut(id, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return source.Details{}, fmt.Errorf("%q is not an owner/repo", id)
	}
	u := strings.TrimRight(cmp.Or(d.URL, apiURL), "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/readme"
	body, _, err := d.do(ctx, call{Method: http.MethodGet, URL: u, Version: mortarVersion, Accept: "application/vnd.github.raw", Auth: true})
	out := source.Details{Versions: []string{}, Dependencies: []string{}, Categories: []string{}}
	if errors.Is(err, httpkit.ErrNotFound) {
		return out, nil
	}
	if err != nil {
		return source.Details{}, answered("github", err)
	}
	out.Description = string(body)
	return out, nil
}
