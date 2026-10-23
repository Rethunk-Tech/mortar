package github

import (
	"context"
	"strings"

	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// CachedFacts answers for each repository ("owner/repo") in installed whose releases list the update check stored,
// however old it is, and never asks GitHub. Released is the installed tag's publish date, Latest the release the
// update check would offer and LatestReleased its date. Downloads sums the assets' download counts, and only when the
// stored list holds every release the repository has. GitHub's repository figures (stars, last push, topics) are not
// in a releases list, so Updated, Rating and Categories stay empty.
func (d *Driver) CachedFacts(_ context.Context, _ string, installed map[string]string) (map[string]source.Facts, error) {
	client := &ghauth.Client{CacheDir: d.CacheDir}
	out := map[string]source.Facts{}
	for repo, version := range installed {
		owner, name, ok := strings.Cut(repo, "/")
		if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
			continue
		}
		all, whole, found := client.CachedReleases(owner, name)
		if !found {
			continue
		}
		var f source.Facts
		if r, _, err := ghauth.Select(all, "", false); err == nil {
			f.Latest, f.LatestReleased = trimTag(r.Tag), r.Published
		}
		want := trimTag(version)
		for _, r := range all {
			if !r.Draft && want != "" && trimTag(r.Tag) == want {
				f.Released = r.Published
			}
			if whole && !r.Draft {
				for _, a := range r.Assets {
					f.Downloads += a.Downloads
				}
			}
		}
		out[repo] = f
	}
	return out, nil
}

func trimTag(tag string) string { return strings.TrimLeft(tag, "vV") }

var _ source.CachedFacter = (*Driver)(nil)
