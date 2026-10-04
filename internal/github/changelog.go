package github

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// MaxChangelog bounds how many releases ReleasesBetween returns.
const MaxChangelog = 10

// ReleasesBetween returns owner/repo's published releases strictly newer than installed and not newer than latest,
// newest first, at most MaxChangelog. A tag that is not a version is skipped. It reads the cached releases list.
func (c *Client) ReleasesBetween(ctx context.Context, owner, repo, installed, latest string) ([]Release, error) {
	all, err := c.Releases(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	var out []Release
	for _, r := range all {
		if r.Draft || len(out) == MaxChangelog {
			continue
		}
		v := tagVersion(r.Tag)
		if c, ok := meta.CompareVersions(v, installed); !ok || c <= 0 {
			continue
		}
		if c, ok := meta.CompareVersions(v, latest); latest != "" && ok && c > 0 {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
