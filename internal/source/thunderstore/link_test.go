package thunderstore

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestParseLink(t *testing.T) {
	r, err := ParseLink("ror2mm://v1/install/thunderstore.io/Alice/MoreCompany/2.0.0/")
	if err != nil || r != (Ref{"Alice", "MoreCompany", "2.0.0"}) {
		t.Fatalf("%+v %v", r, err)
	}
	for _, bad := range []string{
		"https://thunderstore.io/x", "ror2mm://v2/install/thunderstore.io/A/B/1.0.0/", "ror2mm://v1/install/evil.io/A/B/1.0.0/",
		"ror2mm://v1/install/thunderstore.io/A/B/latest/", "ror2mm://v1/install/thunderstore.io/A/../1.0.0/", "ror2mm://v1/install/thunderstore.io/A/B/1.0.0/?x=1",
	} {
		if _, err := ParseLink(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestResolve(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	r, err := d.Resolve(t.Context(), "lethal-company", "alice", "MoreCompany", "2.0.0", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != f.srv.URL+"/package/download/Alice/MoreCompany/2.0.0/" || r.Size != 99 || len(r.Dependencies) != 1 ||
		r.Dependencies[0] != "BepInEx-BepInExPack-5.4.2100" {
		t.Fatalf("%+v", r)
	}
	if r, _ = d.Resolve(t.Context(), "lethal-company", "Alice", "MoreCompany", "", "1.2.3"); r.Version != "2.0.0" {
		t.Fatalf("latest %+v", r)
	}
	if _, err := d.Resolve(t.Context(), "lethal-company", "Eve", "OldMod", "9.9.9", "1.2.3"); err == nil {
		t.Fatal("missing version resolved")
	}
	if _, err := d.Resolve(t.Context(), "lethal-company", "Eve", "OldMod", "2.0.0", "1.2.3"); err != nil {
		t.Fatalf("deprecated package must still resolve as a dependency: %v", err)
	}
}

func TestDependenciesListsPackageIdsOfEachVersion(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0, custom("Ns", "A", v("2.0.0", "Ns-B-1.0.0", "BepInEx-BepInExPack-5.4.2100"), v("1.0.0")))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	two, one, ghost := source.VersionRef{ID: "ns-a", Version: "2.0.0"}, source.VersionRef{ID: "Ns-A", Version: "1.0.0"}, source.VersionRef{ID: "Ns-A", Version: "9"}
	got, err := d.Dependencies(t.Context(), "lethal-company", "1.2.3", []source.VersionRef{two, one, ghost})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Ns-B", "BepInEx-BepInExPack"}; !slices.Equal(got[two], want) {
		t.Errorf("2.0.0 deps = %v, want %v", got[two], want)
	}
	if deps, ok := got[one]; !ok || len(deps) != 0 {
		t.Errorf("1.0.0 deps = %v (listed %v), want an empty listed entry", deps, ok)
	}
	if _, ok := got[ghost]; ok {
		t.Error("a version the index lacks was listed")
	}
}
