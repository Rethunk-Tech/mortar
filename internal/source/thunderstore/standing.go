package thunderstore

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

var _ source.StandingChecker = Driver{}

// Standings flags the packages the community's index no longer lists. An id is "community/Namespace-Name" because the
// index is per community; the answer is keyed by the id as given. A deprecated package is still listed (Deprecated
// reports those), so only a taken-down one is "removed". An empty or unreadable index flags nothing.
func (d Driver) Standings(ctx context.Context, version string, ids []string) (map[string]source.Standing, error) {
	byKey := map[string][]string{}
	for _, id := range ids {
		if key, name, ok := strings.Cut(id, "/"); ok {
			byKey[key] = append(byKey[key], name)
		}
	}
	out := map[string]source.Standing{}
	for key, names := range byKey {
		path, err := d.listingFile(ctx, key, source.UserAgentWithSite(version))
		if err != nil {
			return nil, err
		}
		want := make(map[string]bool, len(names))
		for _, name := range names {
			want[strings.ToLower(name)] = true
		}
		// The problems check asks at every start, so unless Browse already holds the listing it is read a package at a
		// time rather than loaded and kept for listingIdle.
		listed, seen := map[string]bool{}, false
		note := func(p *pkg) {
			seen = true
			if id := packageID(p.Owner, p.Name); want[id] {
				listed[id] = true
			}
		}
		if pk, ok := memoized(key, path); ok {
			for i := range pk {
				note(&pk[i])
			}
		} else if err := eachListed(path, note); err != nil {
			return nil, err
		}
		if !seen {
			continue
		}
		for _, name := range names {
			if !listed[strings.ToLower(name)] {
				out[key+"/"+name] = source.Standing{State: "removed"}
			}
		}
	}
	return out, nil
}
