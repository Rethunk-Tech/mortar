package github

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// maxNotes bounds a release body read from the API.
const maxNotes = 1 << 20

// ReleaseNotes returns the markdown body of owner/repo's release tagged tag. A published release never changes
// its notes, so a cached answer is final and never refetched; a failed lookup is not cached.
func (c *Client) ReleaseNotes(ctx context.Context, owner, repo, tag string) (string, error) {
	name := Key(owner, repo, tag, "") + "-notes-v2.json"
	if notes, ok := meta.Peek[string](c.cache(), name); ok {
		return notes, nil
	}
	base := cmp.Or(c.APIBase, apiBase)
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/repos/"+owner+"/"+repo+"/releases/tags/"+tag, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	DefaultAuth.Apply(req)
	hc := noDowngrade(cmp.Or(c.HTTP, http.DefaultClient))
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	noteRate(resp)
	if err := RateLimited(resp); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub's release API answered %s", resp.Status)
	}
	var rel struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(safefs.CapReader(resp.Body, maxNotes)).Decode(&rel); err != nil {
		return "", fmt.Errorf("read GitHub's release response: %w", err)
	}
	notes := strings.TrimSpace(rel.Body)
	meta.Put(c.cache(), name, notes)
	return notes, nil
}
