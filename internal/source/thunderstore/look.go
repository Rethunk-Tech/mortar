package thunderstore

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Look is how the Mods tab shows a package: its icon's address and its category.
type Look struct {
	Icon     string
	Category string
}

// broadCategories fit nearly every package of a community, or say where it runs rather than what it is.
var broadCategories = []string{"mods", "bepinex", "melonloader", "client-side", "server-side", "ai generated"}

// Category is the one of a package's site categories that says what it is: the first that is not a broad one, else
// the first.
func Category(cats []string) string {
	for _, c := range cats {
		if !slices.Contains(broadCategories, strings.ToLower(c)) {
			return c
		}
	}
	if len(cats) > 0 {
		return cats[0]
	}
	return ""
}

// lookedUp is what CachedLooks already asked of one listing: the listing's index hash and the names it looked up.
type lookedUp struct {
	Hash  string   `json:"hash"`
	Names []string `json:"names"`
}

// CachedLooks returns the Look of each package named (Namespace-Name, any case) that the community's cached listing
// holds, keyed by the lower-case name. It never asks the network, however old the listing is. Names already looked up
// in this listing are not looked up again: a package added from disk is never in it, and decoding the listing costs
// most of a CPU second.
func (d Driver) CachedLooks(key string, names []string) (map[string]Look, error) {
	if !communityKey.MatchString(key) {
		return nil, errors.New("not a Thunderstore community key")
	}
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	var meta cacheMeta
	b, err := fsx.ReadFile(filepath.Join(dir, key+".meta.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &meta); err != nil || meta.Hash == "" {
		return nil, errors.New("no cached Thunderstore listing")
	}
	markPath := filepath.Join(dir, key+".looked.json")
	var mark lookedUp
	if b, err := fsx.ReadFile(markPath); err == nil {
		_ = json.Unmarshal(b, &mark)
	}
	if mark.Hash != meta.Hash {
		mark = lookedUp{Hash: meta.Hash}
	}
	want := map[string]bool{}
	for _, n := range names {
		if id := strings.ToLower(n); !slices.Contains(mark.Names, id) {
			want[id] = true
		}
	}
	if len(want) == 0 {
		return map[string]Look{}, nil
	}
	list, err := loadPackages(key, filepath.Join(dir, key+"-c2-"+meta.Hash+".json"))
	if err != nil {
		return nil, err
	}
	out := map[string]Look{}
	for _, p := range list {
		if id := packageID(p.Owner, p.Name); want[id] {
			out[id] = Look{Icon: p.Icon, Category: Category(p.Categories)}
		}
	}
	for id := range want {
		mark.Names = append(mark.Names, id)
	}
	slices.Sort(mark.Names)
	if b, err := json.Marshal(mark); err == nil {
		_ = datadir.WriteFile(markPath, b, 0o644)
	}
	return out, nil
}
