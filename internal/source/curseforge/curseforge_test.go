package curseforge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	modOpen = `{"data":{"id":10,"name":"Open Mod","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/open"},"allowModDistribution":true}}`
	modShut = `{"data":{"id":20,"name":"Shut Mod","links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/shut"},"allowModDistribution":false}}`
	file2   = `{"id":102,"modId":10,"isAvailable":true,"displayName":"Open 2.0.zip","fileName":"open-2.0.zip","releaseType":1,"fileDate":"2026-02-01T00:00:00Z","fileLength":2048,"downloadUrl":"https://edge/open-2.0.zip","hashes":[{"value":"AB","algo":1},{"value":"cd","algo":2}],"dependencies":[{"modId":11,"relationType":3},{"modId":12,"relationType":2},{"modId":13,"relationType":5}]}`
	file1   = `{"id":101,"modId":10,"isAvailable":true,"displayName":"Open 1.0.zip","fileName":"open-1.0.zip","releaseType":1,"fileDate":"2026-01-01T00:00:00Z","downloadUrl":"","hashes":[]}`
	beta    = `{"id":103,"modId":10,"isAvailable":true,"displayName":"Open 3.0b.zip","fileName":"open-3.0b.zip","releaseType":2,"fileDate":"2026-03-01T00:00:00Z","downloadUrl":"https://edge/b.zip"}`
)

func fake(t *testing.T) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			t.Errorf("key header %q", r.Header.Get("X-Api-Key"))
		}
		switch r.URL.Path {
		case "/mods/search":
			q := r.URL.Query()
			if q.Get("gameId") != "669" || q.Get("classId") != "4643" || q.Get("searchFilter") != "cp" ||
				q.Get("sortField") != "6" || q.Get("index") != "20" || q.Get("pageSize") != "20" {
				t.Errorf("search params %v", q)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":10,"name":"Open Mod","summary":"s","downloadCount":9,"thumbsUpCount":3,"dateModified":"2026-01-01T00:00:00Z","mainFileId":102,"links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/open","sourceUrl":"https://github.com/me/open"},"authors":[{"name":"me"}],"logo":{"thumbnailUrl":"https://img/t.png"},"latestFiles":[` + file2 + `]},{"id":20,"name":"Shut Mod","allowModDistribution":false,"links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/shut"}}],"pagination":{"totalCount":41}}`))
		case "/mods/10":
			_, _ = w.Write([]byte(modOpen))
		case "/mods/20":
			_, _ = w.Write([]byte(modShut))
		case "/mods/10/files":
			_, _ = w.Write([]byte(`{"data":[` + file1 + `,` + file2 + `,` + beta + `]}`))
		case "/mods/20/files":
			_, _ = w.Write([]byte(`{"data":[{"id":201,"modId":20,"isAvailable":true,"displayName":"Shut","fileName":"s.zip","releaseType":1,"downloadUrl":null}]}`))
		case "/mods/10/files/101":
			_, _ = w.Write([]byte(`{"data":` + file1 + `}`))
		case "/mods/10/files/101/download-url":
			http.Error(w, "no", http.StatusForbidden)
		case "/mods":
			_, _ = w.Write([]byte(`{"data":[{"id":10,"links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/open"},"latestFiles":[` + file1 + `,` + file2 + `,` + beta + `]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "4643", GameID: 669}, true
		},
	}
}

func TestSearchMapsHitsAndPages(t *testing.T) {
	t.Parallel()
	page, err := fake(t).Search(context.Background(), source.Query{Game: "stardew", Key: "4643", Text: "cp", Page: 2, Sort: source.SortDownloads})
	if err != nil || page.Total != 41 || len(page.Items) != 2 {
		t.Fatalf("%+v %v", page, err)
	}
	it := page.Items[0]
	if it.ID != "10" || it.Author != "me" || it.Repo != "me/open" || it.Version != "Open 2.0" || it.Endorsements != 3 || it.Downloads != 9 {
		t.Fatalf("%+v", it)
	}
	if it.External || !page.Items[1].External {
		t.Fatalf("external flags: %+v %+v", it, page.Items[1])
	}
	deep, err := fake(t).Search(context.Background(), source.Query{Game: "stardew", Key: "4643", Page: 600})
	if err != nil || len(deep.Items) != 0 {
		t.Fatalf("past the window: %+v %v", deep, err)
	}
}

func TestResolvePicksStableAndFollowsRequiredOnly(t *testing.T) {
	t.Parallel()
	d := fake(t)
	r, err := d.Resolve(context.Background(), "10", "")
	if err != nil || r.URL != "https://edge/open-2.0.zip" || r.Digest != "sha1:ab" || r.Version != "Open 2.0" || r.VersionID != "102" ||
		r.Size != 2048 || r.Name != "Open Mod" || strings.Join(r.Dependencies, ",") != "11" {
		t.Fatalf("%+v %v", r, err)
	}
	if r, err = d.Resolve(context.Background(), "10", "Open 2.0"); err != nil || r.VersionID != "102" {
		t.Fatalf("by name: %+v %v", r, err)
	}
	if _, err = d.Resolve(context.Background(), "10", "Nope"); err == nil {
		t.Fatal("unknown version resolved")
	}
}

func TestForbiddenFilesAreHandedOff(t *testing.T) {
	t.Parallel()
	d := fake(t)
	for _, c := range []struct{ id, version string }{{"20", ""}, {"10", "101"}} {
		_, err := d.Resolve(context.Background(), c.id, c.version)
		manual, ok := errors.AsType[*NotDistributableError](err)
		if !ok || !strings.HasPrefix(manual.PageURL, "https://www.curseforge.com/stardewvalley/mods/") {
			t.Fatalf("mod %s: %v", c.id, err)
		}
	}
}

func TestLatestSkipsInstalledAndBeta(t *testing.T) {
	t.Parallel()
	got, err := fake(t).Latest(context.Background(), components.GameSource{}, "", []source.InstalledFile{
		{ID: "10", Version: "Open 1.0", Digest: "sha1:old"},
		{ID: "10", Version: "Open 2.0", Digest: "sha1:ab"},
	})
	if err != nil || len(got) != 1 || got["sha1:old"].VersionID != "102" || got["sha1:old"].Version != "Open 2.0" {
		t.Fatalf("%+v %v", got, err)
	}
	deps, _ := Driver{}.Dependencies(context.Background(), "", "", []source.VersionRef{{ID: "10", Version: "Open 2.0"}})
	if strings.Join(deps[source.VersionRef{ID: "10", Version: "Open 2.0"}], ",") != "11" {
		t.Fatalf("%v", deps)
	}
}

func TestDetailsAndLinks(t *testing.T) {
	t.Parallel()
	det, err := fake(t).Details(context.Background(), "4643", "10", "")
	if err != nil || strings.Join(det.Versions, ",") != "Open 3.0b,Open 2.0,Open 1.0" || strings.Join(det.Dependencies, ",") != "11" {
		t.Fatalf("%+v %v", det, err)
	}
	if got := (Driver{}).ModPageURL("", "10"); got != "https://www.curseforge.com/projects/10" {
		t.Fatal(got)
	}
}

func TestWithoutAKeyTheSourceIsUnavailable(t *testing.T) {
	t.Setenv(envKey, "")
	d := Driver{}
	if d.Unavailable() == "" || source.Unavailable(d) == "" {
		t.Fatal("a build without a key must report itself unavailable")
	}
	if _, err := d.Search(context.Background(), source.Query{Game: "g", Key: "1"}); err == nil {
		t.Fatal("search without a key succeeded")
	}
	t.Setenv(envKey, "x")
	if d.Unavailable() != "" {
		t.Fatal("the environment key was ignored")
	}
}

// TestLiveStardew runs against the real API when a dev key is set; it is the one check that the fakes' shapes are
// the site's.
func TestLiveStardew(t *testing.T) {
	if os.Getenv(envKey) == "" {
		t.Skip(envKey + " is not set")
	}
	d := Driver{GameSource: func(string) (components.GameSource, bool) {
		return components.GameSource{ID: "curseforge", Key: "4643", GameID: 669}, true
	}}
	page, err := d.Search(t.Context(), source.Query{Game: "stardew", Key: "4643", Text: "Content Patcher", Page: 1})
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("%+v %v", page, err)
	}
	for _, it := range page.Items {
		r, err := d.Resolve(t.Context(), it.ID, "")
		if _, manual := errors.AsType[*NotDistributableError](err); manual {
			continue
		}
		if err != nil || r.URL == "" || r.FileName == "" {
			t.Fatalf("%s: %+v %v", it.Name, r, err)
		}
		return
	}
	t.Skip("no distributable hit on the first page")
}
