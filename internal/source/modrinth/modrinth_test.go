package modrinth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func fake(t *testing.T) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.UserAgent(); !strings.HasPrefix(ua, "Rethunk-Tech/mortar/") || !strings.HasSuffix(ua, " (+https://mortar.rethunk.tech)") {
			t.Errorf("user agent %q", r.UserAgent())
		}
		switch r.URL.Path {
		case "/search":
			q := r.URL.Query()
			if q.Get("facets") != `[["categories:fabric"],["categories:optimization"],["categories!=quilt"]]` ||
				q.Get("index") != "downloads" || q.Get("offset") != "20" || q.Get("query") != "sod" {
				t.Errorf("search params %v", q)
			}
			_, _ = w.Write([]byte(`{"total_hits":41,"hits":[{"project_id":"AANobbMI","slug":"sodium","title":"Sodium","description":"fast","author":"jelly","downloads":9,"follows":3,"date_modified":"2026-09-20T00:00:00Z"}]}`))
		case "/project/AANobbMI/version":
			_, _ = w.Write([]byte(`[{"id":"v2","project_id":"AANobbMI","version_number":"2.0","files":[{"hashes":{"sha512":"ab"},"url":"https://cdn/x.jar","filename":"x.jar","primary":false,"size":1},{"hashes":{"sha512":"cd"},"url":"https://cdn/y.jar","filename":"y.jar","primary":true,"size":7}]},{"id":"v1","project_id":"AANobbMI","version_number":"1.0","files":[{"hashes":{},"url":"https://cdn/o.jar","filename":"o.jar","size":2}]}]`))
		case "/tag/category":
			_, _ = w.Write([]byte(`[{"name":"fabric"},{"name":"Fabric"},{"name":"adventure"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{URL: srv.URL}
}

func TestSearchBuildsFacetsAndMapsHits(t *testing.T) {
	t.Parallel()
	page, err := fake(t).Search(context.Background(), source.Query{
		Key: "fabric", Text: "sod", Page: 2, Version: "1.2", Sort: source.SortDownloads,
		Categories: []string{"Optimization"}, ExcludeCategories: []string{"quilt"},
	})
	if err != nil || page.Total != 41 || len(page.Items) != 1 {
		t.Fatalf("%+v %v", page, err)
	}
	it := page.Items[0]
	if it.ID != "AANobbMI" || it.URL != "https://modrinth.com/project/sodium" || it.Endorsements != 3 || it.Downloads != 9 {
		t.Fatalf("%+v", it)
	}
}

func TestResolveAndVersions(t *testing.T) {
	t.Parallel()
	d := fake(t)
	r, err := d.Resolve(context.Background(), "AANobbMI", "", "1.2")
	if err != nil || r.URL != "https://cdn/y.jar" || r.Digest != "sha512:cd" || r.Size != 7 || r.Version != "2.0" {
		t.Fatalf("%+v %v", r, err)
	}
	if r, err = d.Resolve(context.Background(), "AANobbMI", "1.0", "1.2"); err != nil || r.Digest != "" || r.FileName != "o.jar" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err = d.Resolve(context.Background(), "AANobbMI", "9", "1.2"); err == nil {
		t.Fatal("unknown version resolved")
	}
	if v, _ := d.Versions(context.Background(), "AANobbMI", "1.2"); strings.Join(v, ",") != "2.0,1.0" {
		t.Fatalf("%v", v)
	}
}

func TestCategoriesAndLinks(t *testing.T) {
	t.Parallel()
	got, err := fake(t).Categories(context.Background(), "")
	if err != nil || strings.Join(got, ",") != "adventure,fabric" {
		t.Fatalf("%v %v", got, err)
	}
	if facet("project_type:mod") != "project_type:mod" || (Driver{}).ModPageURL("", "") != "" {
		t.Fatal("facet or link")
	}
}
