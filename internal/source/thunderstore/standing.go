package thunderstore

import (
	"context"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
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
	c, err := d.client(version)
	if err != nil {
		return nil, err
	}
	out := map[string]source.Standing{}
	for key, names := range byKey {
		want := make(map[string]bool, len(names))
		for _, name := range names {
			want[strings.ToLower(name)] = true
		}
		// The problems check asks at every start, so unless Browse already holds the listing it is read a package at a
		// time rather than loaded and kept in memory.
		listed, seen := map[string]bool{}, false
		note := func(p *tstore.Package) {
			seen = true
			if id := p.ID(); want[id] {
				listed[id] = true
			}
		}
		if err := c.Each(ctx, key, note); err != nil {
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
