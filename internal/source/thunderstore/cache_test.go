package thunderstore

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// testdata/listing-c4 is the cache folder Mortar wrote before the index moved to go-thunderstore: the c4 listing, its
// meta file and the versions side file, as that code built them from this package's fake site. Players upgrading keep
// it, so it must answer from disk with no request.
func TestTheC4CacheAnEarlierReleaseWroteIsReadWithoutTheNetwork(t *testing.T) {
	cache := t.TempDir()
	if err := os.CopyFS(cache, os.DirFS("testdata/listing-c4")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/listing-c4/thunderstore/lethal-company.meta.json")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Fetched time.Time `json:"fetched"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	var asked atomic.Int32
	offline := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		asked.Add(1)
		return nil, errors.New("offline")
	})}
	d := Driver{HTTP: offline, CacheDir: cache, Now: func() time.Time { return meta.Fetched.Add(time.Minute) }}
	t.Cleanup(func() { restart(d) })

	facts, err := d.CachedFacts(t.Context(), "lethal-company", map[string]string{"alice-morecompany": "1.0.0"})
	if f := facts["alice-morecompany"]; err != nil || f.Latest != "2.0.0" || f.Downloads != 501 || f.Released != "2025-12-01T00:00:00Z" {
		t.Fatalf("stored facts = %+v, %v", facts, err)
	}
	p, err := d.Search(t.Context(), source.Query{Key: "lethal-company", Text: "company", Page: 1, Version: "1.2.3"})
	if err != nil || p.Total != 3 || p.Items[0].Name != "MoreCompany" {
		t.Fatalf("search = %v total %d, %v", names(p), p.Total, err)
	}
	ref := source.VersionRef{ID: "Alice-MoreCompany", Version: "2.0.0"}
	deps, err := d.Dependencies(t.Context(), "lethal-company", "1.2.3", []source.VersionRef{ref})
	if err != nil || !slices.Equal(deps[ref], []string{"BepInEx-BepInExPack"}) {
		t.Fatalf("dependencies = %v, %v", deps, err)
	}
	if v, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3"); err != nil || !slices.Equal(v, []string{"2.0.0", "1.0.0"}) {
		t.Fatalf("versions = %v, %v", v, err)
	}
	if n := asked.Load(); n != 0 {
		t.Fatalf("reading the stored cache sent %d requests", n)
	}
}
