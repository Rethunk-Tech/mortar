package github

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// maxNotes bounds a release body read from the API.
const maxNotes = 1 << 20

type notesEntry struct {
	Notes string `json:"notes"`
}

// ReleaseNotes returns the markdown body of owner/repo's release tagged tag. A published release never changes
// its notes, so a cached answer is final and never refetched; a failed lookup is not cached.
func (c *Client) ReleaseNotes(ctx context.Context, owner, repo, tag string) (string, error) {
	dir, err := c.cacheDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, Key(owner, repo, tag, "")+"-notes.json")
	if b, err := fsx.ReadFile(path); err == nil {
		var hit notesEntry
		if json.Unmarshal(b, &hit) == nil {
			return hit.Notes, nil
		}
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
	hc := cmp.Or(c.HTTP, http.DefaultClient)
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := RateLimited(resp); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub's release API answered %s", resp.Status)
	}
	var rel struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxNotes)).Decode(&rel); err != nil {
		return "", fmt.Errorf("read GitHub's release response: %w", err)
	}
	notes := strings.TrimSpace(rel.Body)
	if os.MkdirAll(dir, 0o700) == nil {
		_ = datadir.WriteJSON(path, notesEntry{Notes: notes})
	}
	return notes, nil
}
