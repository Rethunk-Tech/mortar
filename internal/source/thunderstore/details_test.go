package thunderstore

import (
	"net/http"
	"slices"
	"testing"
	"time"
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
		!slices.Equal(got.Released, []int64{1768471200, 1764547200}) ||
		!slices.Equal(got.Dependencies, []string{"BepInEx-BepInExPack-5.4.2100"}) || !slices.Equal(got.Categories, []string{"Items"}) {
		t.Fatalf("details = %+v", got)
	}
	if _, err := d.Details(t.Context(), "lethal-company", "Nobody-Nothing", "1.2.3"); err == nil {
		t.Fatal("a package missing from the index answered")
	}
}

func TestChangelogReadsTheUpdatedVersionsFile(t *testing.T) {
	f := newFake(t)
	f.mux.HandleFunc("/api/experimental/package/Alice/MoreCompany/1.0.0/changelog/",
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"markdown":"- fixed"}`)) })
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Changelog(t.Context(), "lethal-company", "Alice-MoreCompany", "0.9.0", "1.0.0", "1.2.3")
	if err != nil || len(got) != 1 || got[0].Version != "1.0.0" || got[0].Body != "- fixed" {
		t.Fatalf("notes = %+v, err %v", got, err)
	}
	if none, err := d.Changelog(t.Context(), "lethal-company", "Alice-MoreCompany", "1.0.0", "2.0.0", "1.2.3"); err != nil || len(none) != 0 {
		t.Fatalf("a version without a CHANGELOG gave %+v, err %v", none, err)
	}
}

func TestReportTargetIsThePackagePage(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.ReportTarget(t.Context(), "lethal-company", "alice-morecompany", "1.2.3")
	if err != nil || got.URL == "" {
		t.Fatalf("target %+v, %v", got, err)
	}
	if _, err := d.ReportTarget(t.Context(), "lethal-company", "Nobody-Nothing", "1.2.3"); err == nil {
		t.Fatal("a package missing from the index answered")
	}
}

func TestDetailsCarryOnlyAWebDonationLink(t *testing.T) {
	f := newFake(t)
	f.chunk0[0]["donation_link"] = "https://ko-fi.com/alice"
	f.chunk0[1]["donation_link"] = "javascript:alert(1)"
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	for id, want := range map[string]string{"Alice-MoreCompany": "https://ko-fi.com/alice", "Bob-Cheaty": ""} {
		got, err := d.Details(t.Context(), "lethal-company", id, "1.2.3")
		if err != nil || got.Donation != want {
			t.Errorf("%s: donation %q, err %v, want %q", id, got.Donation, err, want)
		}
	}
}

func TestDetailsAsksTheReadmeAndTheChangelogTogether(t *testing.T) {
	f := newFake(t)
	readmeIn, changelogIn := make(chan struct{}), make(chan struct{})
	serve := func(mine, other chan struct{}, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			close(mine)
			select {
			case <-other:
				_, _ = w.Write([]byte(body))
			case <-time.After(5 * time.Second):
				http.Error(w, "the other document was never asked for while this one was open", http.StatusGatewayTimeout)
			}
		}
	}
	f.mux.HandleFunc("/api/experimental/package/Alice/MoreCompany/2.0.0/readme/", serve(readmeIn, changelogIn, `{"markdown":"# More"}`))
	f.mux.HandleFunc("/api/experimental/package/Alice/MoreCompany/2.0.0/changelog/", serve(changelogIn, readmeIn, `{"markdown":"- fixed"}`))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Details(t.Context(), "lethal-company", "Alice-MoreCompany", "1.2.3")
	if err != nil || got.Description != "# More" || got.Changelog != "- fixed" {
		t.Fatalf("details = %+v, err %v", got, err)
	}
}
