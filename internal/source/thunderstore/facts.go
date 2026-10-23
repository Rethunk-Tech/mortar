package thunderstore

import (
	"slices"
	"strings"
	"time"
)

// Fact is what the community's listing says of one package and of the version of it that is installed.
type Fact struct {
	// Released is when the installed version was published (RFC 3339), empty when the listing lacks that version or its date.
	Released string
	// Latest is the newest version, the one the update check offers, and LatestReleased its date.
	Latest, LatestReleased string
	// Updated is the package's date_updated as the site gives it.
	Updated string
	Rating  int
	// Downloads is the package's total over all its versions, the count the site's package page shows.
	Downloads  int
	Categories []string
}

func released(t int64) string {
	if t == 0 {
		return ""
	}
	return time.Unix(t, 0).UTC().Format(time.RFC3339)
}

// CachedFacts returns the Fact of each package in installed (lower-case Namespace-Name to the installed version) that
// the community's cached listing holds, keyed by the same name. It never asks the network, however old the listing is:
// a listing Browse holds in memory is read as is, else the stored file is read a package at a time and not kept.
func (d Driver) CachedFacts(key string, installed map[string]string) (map[string]Fact, error) {
	dir, meta, err := d.cachedMeta(key)
	if err != nil {
		return nil, err
	}
	path := listingPath(dir, key, meta.Hash)
	found := make(map[string]pkg, len(installed))
	keep := func(p *pkg) {
		id := packageID(p.Owner, p.Name)
		if _, want := installed[id]; !want {
			return
		}
		if o, seen := found[id]; !seen || !keepsOver(o, *p) {
			found[id] = *p
		}
	}
	if pk, ok := memoized(key, path); ok {
		for i := range pk {
			keep(&pk[i])
		}
	} else if err := eachListed(path, keep); err != nil {
		return nil, err
	}
	out := make(map[string]Fact, len(found))
	for id, p := range found {
		f := Fact{
			Latest: p.Versions[0].Number, LatestReleased: released(p.Versions[0].Created),
			Updated: p.Updated, Rating: p.Rating, Downloads: p.Downloads, Categories: slices.Clone(p.Categories),
		}
		if i := slices.IndexFunc(p.Versions, func(v version) bool { return v.Number == strings.TrimLeft(installed[id], "vV") }); i >= 0 {
			f.Released = released(p.Versions[i].Created)
		}
		out[id] = f
	}
	return out, nil
}
