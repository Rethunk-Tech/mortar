package thunderstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestItemsOnAColdCacheLeaveNoListingInMemory(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Items(t.Context(), "lethal-company", "1.2.3", []string{"alice-morecompany"})
	if err != nil || got["alice-morecompany"].Name != "MoreCompany" {
		t.Fatalf("items = %v, %v", got, err)
	}
	if held(t, d) {
		t.Fatal("an item lookup kept the whole listing in memory")
	}
}

func TestCategoriesAndDeprecatedAnswerFromTheDerivedFileWithoutTheListing(t *testing.T) {
	f := newFake(t)
	cache := t.TempDir()
	d := Driver{URL: f.srv.URL, CacheDir: cache}
	if _, err := d.Deprecated(t.Context(), "lethal-company", "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Categories(t.Context(), "lethal-company"); err != nil {
		t.Fatal(err)
	}
	if held(t, d) {
		t.Fatal("a category lookup kept the whole listing in memory")
	}
	dir := filepath.Join(cache, "thunderstore")
	s := readDerived(dir, "lethal-company")
	if s.Listing == "" || len(s.Categories) == 0 || len(s.Deprecation) == 0 {
		t.Fatalf("derived file = %+v", s)
	}
	// Only the file can say this: the listing holds no such category or package.
	s.Categories = []string{"FromTheFile"}
	s.Deprecation = map[string]Deprecation{"x-y": {Replacement: "a-b"}}
	b, _ := json.Marshal(s)
	if err := os.WriteFile(derivedPath(dir, "lethal-company"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	cats, err := d.Categories(t.Context(), "lethal-company")
	if err != nil || !slices.Equal(cats, []string{"FromTheFile"}) {
		t.Fatalf("categories = %v, %v", cats, err)
	}
	dep, err := d.Deprecated(t.Context(), "lethal-company", "1.2.3")
	if err != nil || dep["x-y"].Replacement != "a-b" || len(dep) != 1 {
		t.Fatalf("deprecated = %v, %v", dep, err)
	}
	if held(t, d) {
		t.Fatal("a deprecation lookup kept the whole listing in memory")
	}
}
