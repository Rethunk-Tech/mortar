package github

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
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
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	u := strings.TrimRight(cmp.Or(d.URL, apiURL), "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", source.UserAgent(version))
	req.Header.Set("Accept", "application/vnd.github+json")
	ghauth.DefaultAuth.Apply(req)
	resp, err := cmp.Or(d.HTTP, http.DefaultClient).Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		return false, source.Busy("GitHub", resp)
	default:
		return false, fmt.Errorf("github answered %s", resp.Status)
	}
	var repoInfo struct {
		Archived bool `json:"archived"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(body, &repoInfo); err != nil {
		return false, err
	}
	return repoInfo.Archived, nil
}
