package curseforge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChangelogReadsFilesNewerThanInstalled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/10/files":
			_, _ = w.Write([]byte(`{"data":[` + file1 + `,` + file2 + `,` + beta + `]}`))
		case "/mods/10/files/102/changelog":
			_, _ = w.Write([]byte(`{"data":"<p>Two</p><ul><li>fix</li></ul>"}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	d := Driver{URL: srv.URL, HTTP: srv.Client(), Key: func() string { return "k" }}
	got, err := d.Changelog(t.Context(), "", "10", "Open 1.0", "Open 2.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Version != "Open 2.0" || got[0].Date != "2026-02-01" || got[0].Body != "Two\n\n• fix" {
		t.Fatalf("notes = %+v", got)
	}
	got, err = d.Changelog(t.Context(), "", "10", "Open 1.0", "102", "")
	if err != nil || len(got) != 1 || got[0].Version != "Open 2.0" {
		t.Fatalf("latest file id = %+v, %v", got, err)
	}
}
