package github

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

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

func (d *Driver) archived(ctx context.Context, version, repo string) (bool, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return false, nil
	}
	u := strings.TrimRight(cmp.Or(d.URL, apiURL), "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	body, _, err := d.do(ctx, call{Method: http.MethodGet, URL: u, Version: version, Accept: "application/vnd.github+json", Auth: true})
	if errors.Is(err, httpkit.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, answered("github", err)
	}
	var repoInfo struct {
		Archived bool `json:"archived"`
	}
	if err := json.Unmarshal(body, &repoInfo); err != nil {
		return false, err
	}
	return repoInfo.Archived, nil
}
