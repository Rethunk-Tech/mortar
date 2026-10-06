package thunderstore

import (
	"net/http"
	"slices"
	"testing"
)

func TestDetailsReadsTheIndexAndTheNewestReadme(t *testing.T) {
	f := newFake(t)
	f.mux.HandleFunc("/api/experimental/package/Alice/MoreCompany/2.0.0/readme/",
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"markdown":"# More"}`)) })
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Details(t.Context(), "lethal-company", "Alice-MoreCompany", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "# More" || got.Changelog != "" || !slices.Equal(got.Versions, []string{"2.0.0", "1.0.0"}) ||
		!slices.Equal(got.Dependencies, []string{"BepInEx-BepInExPack-5.4.2100"}) || !slices.Equal(got.Categories, []string{"Items"}) {
		t.Fatalf("details = %+v", got)
	}
	if _, err := d.Details(t.Context(), "lethal-company", "Nobody-Nothing", "1.2.3"); err == nil {
		t.Fatal("a package missing from the index answered")
	}
}
