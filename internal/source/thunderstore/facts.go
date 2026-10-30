package thunderstore

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func released(t int64) string {
	if t == 0 {
		return ""
	}
	return time.Unix(t, 0).UTC().Format(time.RFC3339)
}

// CachedFacts returns the Facts of each package in installed (lower-case Namespace-Name to the installed version) that
// the community's cached listing holds, keyed by the same name. It never asks the network, however old the listing is:
// a listing Browse holds in memory is read as is, else the stored file is read a package at a time and not kept.
func (d Driver) CachedFacts(ctx context.Context, key string, installed map[string]string) (map[string]source.Facts, error) {
	c, err := d.client("")
	if err != nil {
		return nil, err
	}
	found := make(map[string]tstore.Package, len(installed))
	keep := func(p *tstore.Package) {
		id := p.ID()
		if _, want := installed[id]; !want {
			return
		}
		if o, seen := found[id]; !seen || !o.Outranks(*p) {
			found[id] = *p
		}
	}
	if stored, err := c.Stored(ctx, key, keep); err != nil || !stored {
		return nil, cmp.Or(err, errNotStored)
	}
	out := make(map[string]source.Facts, len(found))
	for id, p := range found {
		f := source.Facts{
			Latest: p.Versions[0].Number, LatestReleased: released(p.Versions[0].Created),
			Updated: p.Updated, Rating: p.Rating, Downloads: p.Downloads, Categories: slices.Clone(p.Categories),
		}
		if i := slices.IndexFunc(p.Versions, func(v tstore.Version) bool { return v.Number == strings.TrimLeft(installed[id], "vV") }); i >= 0 {
			f.Released = released(p.Versions[i].Created)
		}
		out[id] = f
	}
	return out, nil
}

var _ source.CachedFacter = Driver{}
