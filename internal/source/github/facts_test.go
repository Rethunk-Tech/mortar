package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
)

const factsReleases = `[
 {"tag_name":"v3.0.0-beta","prerelease":true,"published_at":"2026-03-01T00:00:00Z","assets":[{"name":"m.zip","content_type":"application/zip","download_count":5}]},
 {"tag_name":"v2.0.0","published_at":"2026-02-01T00:00:00Z","assets":[{"name":"m.zip","content_type":"application/zip","download_count":100},{"name":"m.7z","download_count":20}]},
 {"tag_name":"v1.0.0","published_at":"2025-12-01T00:00:00Z","assets":[{"name":"m.zip","content_type":"application/zip","download_count":30}]}
]`

func TestFactsComeFromTheCachedReleasesAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(factsReleases)) }))
	dir := t.TempDir()
	d := &Driver{CacheDir: dir}
	installed := map[string]string{"owner/repo": "v1.0.0", "owner/other": "1.0.0"}
	if got, err := d.CachedFacts(t.Context(), "", installed); err != nil || len(got) != 0 {
		t.Fatalf("a repository with no stored releases answered: %+v %v", got, err)
	}
	if _, err := (&ghauth.Client{APIBase: srv.URL, CacheDir: dir}).Releases(t.Context(), "owner", "repo"); err != nil {
		t.Fatal(err)
	}
	srv.Close()
	got, err := d.CachedFacts(t.Context(), "", installed)
	if err != nil || len(got) != 1 {
		t.Fatalf("facts %+v %v", got, err)
	}
	f := got["owner/repo"]
	if f.Released != "2025-12-01T00:00:00Z" || f.Latest != "2.0.0" || f.LatestReleased != "2026-02-01T00:00:00Z" || f.Downloads != 155 ||
		f.Updated != "" || f.Rating != 0 || len(f.Categories) != 0 {
		t.Fatalf("facts = %+v", f)
	}
}
