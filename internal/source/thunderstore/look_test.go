package thunderstore

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
)

func TestCategoryPrefersWhatThePackageIs(t *testing.T) {
	for want, cats := range map[string][]string{
		"Tools":     {"Mods", "Tools", "Libraries", "BepInEx"},
		"Libraries": {"BepInEx", "Client-side", "Libraries"},
		"Mods":      {"Mods", "BepInEx"},
		"":          nil,
	} {
		if got := Category(cats); got != want {
			t.Errorf("Category(%v) = %q, want %q", cats, got, want)
		}
	}
}

// A site's categories come back in the site's order; a listing that lacks the package leaves the one recorded at install.
func TestSiteCategoriesKeepTheSitesOrderAndFallBackToTheStoredOne(t *testing.T) {
	for _, tc := range []struct {
		listed []string
		stored string
		want   []string
	}{
		{[]string{"Tools", "Mods", "Libraries"}, "Mods", []string{"Tools", "Mods", "Libraries"}},
		{nil, "Tools", []string{"Tools"}},
		{[]string{}, "Tools", []string{"Tools"}},
		{nil, "", nil},
	} {
		if got := SiteCategories(tc.listed, tc.stored); !slices.Equal(got, tc.want) {
			t.Errorf("SiteCategories(%v, %q) = %v, want %v", tc.listed, tc.stored, got, tc.want)
		}
	}
}

func TestLooksComeFromTheIndexAndTheCacheAlone(t *testing.T) {
	f := newFake(t)
	icon := "https://ccdn.thunderstore.io/live/repository/icons/Ns-Gen-1.0.0.png"
	l := custom("Ns", "Gen", ver{VersionNumber: "1.0.0", Icon: icon})
	l["categories"] = []string{"Mods", "Tools"}
	f.chunk0 = append(f.chunk0, l)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Closure(t.Context(), "lethal-company", []tstore.Ref{{Namespace: "Ns", Name: "Gen", Version: ""}}, "1.2.3", DepsNewest)
	if err != nil || len(got) != 1 || got[0].Look.Category != "Tools" || got[0].Look.Icon != icon ||
		!slices.Equal(got[0].Look.Categories, []string{"Mods", "Tools"}) {
		t.Fatalf("closure %+v %v", got, err)
	}
	f.srv.Close()
	looks, err := d.CachedLooks("lethal-company", []string{"NS-GEN", "Ns-Missing"})
	if err != nil || len(looks) != 1 || looks["ns-gen"].Category != "Tools" || looks["ns-gen"].Icon != icon {
		t.Fatalf("cached looks %+v %v", looks, err)
	}
	// Every category the site lists, in the site's order, beside the one the list groups by.
	if got := looks["ns-gen"].Categories; !slices.Equal(got, []string{"Mods", "Tools"}) {
		t.Fatalf("cached look categories = %v", got)
	}
	for range 2 {
		if icons, err := d.CachedIcons("lethal-company", []string{"NS-GEN", "Ns-Missing"}); err != nil || len(icons) != 1 || icons["ns-gen"] != icon {
			t.Fatalf("cached icons %+v %v", icons, err)
		}
	}
	// A name the listing lacks was looked up once; asking again must not read the listing, so a damaged one goes
	// unnoticed.
	listings, _ := filepath.Glob(filepath.Join(d.CacheDir, "thunderstore", "lethal-company-c*-*.json"))
	if len(listings) != 1 || os.WriteFile(listings[0], []byte("not json"), 0o600) != nil {
		t.Fatal(listings)
	}
	restart(d)
	if looks, err := d.CachedLooks("lethal-company", []string{"ns-missing"}); err != nil || len(looks) != 0 {
		t.Fatalf("a name already looked up read the listing again: %+v %v", looks, err)
	}
	if _, err := d.CachedLooks("lethal-company", []string{"Ns-Other"}); err == nil {
		t.Fatal("a new name did not read the listing")
	}
}
