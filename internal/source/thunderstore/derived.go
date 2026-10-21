package thunderstore

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// What the Problems check and Browse's filters ask of the whole listing (which packages are deprecated, which
// categories exist) is worked out once per stored listing and kept in a small file beside it, so neither asks a
// start to decode tens of megabytes.

type derivedFile struct {
	// Listing is the file name of the stored listing this was worked out from.
	Listing     string                 `json:"listing"`
	Categories  []string               `json:"categories"`
	Deprecation map[string]Deprecation `json:"deprecation"`
}

var derivedMu sync.Mutex

func derivedPath(dir, key string) string { return filepath.Join(dir, key+".derived.json") }

func readDerived(dir, key string) derivedFile {
	var s derivedFile
	if b, err := fsx.ReadFile(derivedPath(dir, key)); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

// derivedFor is the derived file of the stored listing at path, built from it first when it is missing or was built
// from another listing.
func derivedFor(dir, key, path string) (derivedFile, error) {
	derivedMu.Lock()
	defer derivedMu.Unlock()
	stamp := filepath.Base(path)
	if s := readDerived(dir, key); s.Listing == stamp {
		return s, nil
	}
	s, err := deriveFrom(path)
	if err != nil {
		return s, err
	}
	if b, err := json.Marshal(s); err == nil {
		_ = datadir.WriteFile(derivedPath(dir, key), b, 0o644)
	}
	return s, nil
}

// refreshDerived rebuilds the derived file for a freshly built listing, when an earlier listing had one.
func refreshDerived(dir, key, path string) {
	if readDerived(dir, key).Listing != "" {
		_, _ = derivedFor(dir, key, path)
	}
}

// deriveFrom reads the listing a package at a time, holding the one winning listing per id (as dedupeIDs does) with
// only the fields the answers use.
func deriveFrom(path string) (derivedFile, error) {
	winners := map[string]pkg{}
	err := eachListed(path, func(p *pkg) {
		id := packageID(p.Owner, p.Name)
		if o, seen := winners[id]; seen && keepsOver(o, *p) {
			return
		}
		w := pkg{Owner: p.Owner, Name: p.Name, Hidden: p.Hidden, Updated: p.Updated, Categories: p.Categories}
		if p.Hidden {
			w.Summary = p.Summary
		}
		winners[id] = w
	})
	if err != nil {
		return derivedFile{}, err
	}
	var cats []string
	live := map[string]string{}
	for _, p := range winners {
		cats = append(cats, p.Categories...)
		if !p.Hidden {
			live[strings.ToLower(p.Owner+"-"+p.Name)] = p.Owner + "-" + p.Name
		}
	}
	out := map[string]Deprecation{}
	for _, p := range winners {
		if p.Hidden {
			self := strings.ToLower(p.Owner + "-" + p.Name)
			out[self] = Deprecation{Replacement: replacementFor(p, self, live)}
		}
	}
	return derivedFile{Listing: filepath.Base(path), Categories: source.UniqueNames(cats), Deprecation: out}, nil
}
