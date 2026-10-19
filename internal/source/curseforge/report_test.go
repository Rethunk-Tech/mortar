package curseforge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportTargetPrefersTheIssueTrackerOnGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/1":
			_, _ = w.Write([]byte(`{"data":{"id":1,"links":{"websiteUrl":"https://cf/p","issuesUrl":"https://github.com/me/p/issues"}}}`))
		case "/mods/2":
			_, _ = w.Write([]byte(`{"data":{"id":2,"links":{"websiteUrl":"https://cf/q","issuesUrl":"https://tracker.example/q"}}}`))
		case "/mods/3":
			_, _ = w.Write([]byte(`{"data":{"id":3,"links":{"websiteUrl":"https://cf/r","issuesUrl":"http://tracker.example/r"}}}`))
		case "/mods/4":
			_, _ = w.Write([]byte(`{"data":{"id":4,"links":{"websiteUrl":"https://cf/s","issuesUrl":"https://u:p@tracker.example/s","sourceUrl":"https://github.com/a/b%3Fx/"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	d := Driver{URL: srv.URL, HTTP: srv.Client(), Key: func() string { return "k" }}
	if got, err := d.ReportTarget(t.Context(), "", "1", ""); err != nil || got.Repo != "me/p" {
		t.Fatalf("github tracker gave %+v, %v", got, err)
	}
	if got, err := d.ReportTarget(t.Context(), "", "2", ""); err != nil || got.Repo != "" || got.URL != "https://tracker.example/q" {
		t.Fatalf("other tracker gave %+v, %v", got, err)
	}
	for _, id := range []string{"3", "4"} {
		got, err := d.ReportTarget(t.Context(), "", id, "")
		if err != nil || got.Repo != "" || !strings.HasPrefix(got.URL, "https://cf/") {
			t.Fatalf("mod %s gave %+v, %v", id, got, err)
		}
	}
}
