package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
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
			_, _ = w.Write([]byte(`[{"id":"v2","project_id":"AANobbMI","version_number":"2.0","dependencies":[{"project_id":"P1","version_id":"V1","dependency_type":"required"},{"project_id":"P2","dependency_type":"optional"},{"version_id":"V3","dependency_type":"required"}],"files":[{"hashes":{"sha512":"ab"},"url":"https://cdn/x.jar","filename":"x.jar","primary":false,"size":1},{"hashes":{"sha512":"cd"},"url":"https://cdn/y.jar","filename":"y.jar","primary":true,"size":7}]},{"id":"v1","project_id":"AANobbMI","version_number":"1.0","files":[{"hashes":{},"url":"https://cdn/o.jar","filename":"o.jar","size":2}]}]`))
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
	if err != nil || r.URL != "https://cdn/y.jar" || r.Digest != "sha512:cd" || r.Size != 7 || r.Version != "2.0" || len(r.Dependencies) != 1 || r.Dependencies[0] != (Dependency{ProjectID: "P1", VersionID: "V1"}) {
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

// One request finds the newest version of every file for the game's loaders and versions, and a second reads the
// installed versions of the files with a newer one, so both sides' dependencies are known.
func TestLatestChecksEveryFileInOneBatch(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.Method+" "+r.URL.Path)
		var body struct {
			Hashes       []string `json:"hashes"`
			Algorithm    string   `json:"algorithm"`
			Loaders      []string `json:"loaders"`
			GameVersions []string `json:"game_versions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Algorithm != "sha512" {
			t.Errorf("body %+v %v", body, err)
		}
		switch r.URL.Path {
		case "/version_files/update":
			if strings.Join(body.Hashes, ",") != "aa11,bb22" || strings.Join(body.Loaders, ",") != "fabric" || strings.Join(body.GameVersions, ",") != "1.21" {
				t.Errorf("update body %+v", body)
			}
			http.ServeFile(w, r, "testdata/version_files_update.json")
		case "/version_files":
			if strings.Join(body.Hashes, ",") != "aa11" {
				t.Errorf("installed body %+v", body)
			}
			http.ServeFile(w, r, "testdata/version_files.json")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	d := Driver{URL: srv.URL}
	src := components.GameSource{ID: "modrinth", Loaders: []string{"fabric"}, GameVersions: []string{"1.21"}}
	got, err := d.Latest(context.Background(), src, "", []source.InstalledFile{
		{ID: "sodium", Version: "0.5.0", Digest: "sha512:aa11"},
		{ID: "lithium", Version: "1.0", Digest: "sha512:BB22"},
		{ID: "local", Version: "1", Digest: ""},
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("latest = %+v %v", got, err)
	}
	if want := (source.Latest{ProjectID: "AANobbMI", Version: "0.6.0", VersionID: "SoD3", URL: "https://modrinth.com/project/AANobbMI/version/SoD3"}); got["sha512:aa11"] != want {
		t.Errorf("sodium = %+v, want %+v", got["sha512:aa11"], want)
	}
	if strings.Join(asked, ";") != "POST /version_files/update;POST /version_files" {
		t.Errorf("requests = %v", asked)
	}
	before, after := source.VersionRef{ID: "AANobbMI", Version: "0.5.0"}, source.VersionRef{ID: "AANobbMI", Version: "0.6.0"}
	deps, _ := d.Dependencies(context.Background(), "", "", []source.VersionRef{before, after, {ID: "x", Version: "1"}})
	if strings.Join(deps[before], ",") != "P1,P8" || strings.Join(deps[after], ",") != "P1,P9" || len(deps) != 2 {
		t.Errorf("dependencies = %v", deps)
	}
}

func TestLatestSaysBusyWhenRateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	_, err := Driver{URL: srv.URL}.Latest(context.Background(), components.GameSource{}, "", []source.InstalledFile{{Digest: "sha512:aa"}})
	if !errors.Is(err, source.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
}
