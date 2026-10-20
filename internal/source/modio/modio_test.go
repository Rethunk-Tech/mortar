package modio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const timberAPI = `{"id":42,"name":"TimberAPI","summary":"api","profile_url":"https://mod.io/g/timberborn/m/timberapi",
"date_updated":1775230769,"maturity_option":8,"submitted_by":{"username":"eyes"},"logo":{"thumb_320x180":"https://thumb.modcdn.io/l.png"},
"stats":{"downloads_total":125963,"ratings_positive":143},"tags":[{"name":"Mod"}],"description":"<p>Hello <b>you</b></p>",
"modfile":{"id":9,"filesize":2048,"filename":"t.zip","version":"0.8","filehash":{"md5":"ABC"},"download":{"binary_url":"https://g-1.modapi.io/f/9"}}}`

func fake(t *testing.T) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Modio-Platform") != "windows" {
			t.Errorf("platform header %q", r.Header.Get("X-Modio-Platform"))
		}
		q := r.URL.Query()
		switch r.URL.Path {
		case "/me":
			_, _ = w.Write([]byte(`{"username":"nomad"}`))
		case "/games/3659/mods":
			if q.Get("id-in") != "" {
				_, _ = w.Write([]byte(`{"data":[` + timberAPI + `]}`))
				return
			}
			if q.Get("_q") != "timber" || q.Get("_sort") != "-downloads_total" || q.Get("_limit") != "50" || q.Get("_offset") != "50" ||
				q.Get("tags-in") != "Mod" || q.Get("tags-not-in") != "Map" {
				t.Errorf("search params %v", q)
			}
			_, _ = w.Write([]byte(`{"data":[` + timberAPI + `],"result_total":3002}`))
		case "/games/3659/mods/42":
			_, _ = w.Write([]byte(timberAPI))
		case "/games/3659/mods/42/files":
			_, _ = w.Write([]byte(`{"data":[{"id":9,"version":"0.8","changelog":" fixed ","download":{"binary_url":"https://x/9"}},
				{"id":7,"version":"","filename":"old.zip","download":{"binary_url":"https://x/7"},"filehash":{"md5":"def"}}]}`))
		case "/games/3659/mods/42/dependencies":
			_, _ = w.Write([]byte(`{"data":[{"id":5,"name":"Harmony"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{URL: srv.URL, Token: func() (string, error) { return "tok", nil }}
}

func TestSearch(t *testing.T) {
	t.Parallel()
	page, err := fake(t).Search(context.Background(), source.Query{
		Key: "3659", Text: "timber", Page: 2, Size: 50, Sort: source.SortDownloads,
		Categories: []string{"Mod"}, ExcludeCategories: []string{"Map"},
	})
	if err != nil || page.Total != 3002 || len(page.Items) != 1 {
		t.Fatalf("%+v %v", page, err)
	}
	it := page.Items[0]
	want := source.Item{
		Source: "modio", ID: "42", Name: "TimberAPI", Summary: "api", Author: "eyes", Version: "0.8",
		Picture: "https://thumb.modcdn.io/l.png", Endorsements: 143, Downloads: 125963, Updated: "2026-04-03T15:39:29Z",
		Size: 2048, URL: "https://mod.io/g/timberborn/m/timberapi", Adult: true,
	}
	if it.Name != want.Name || it.ID != want.ID || it.Version != want.Version || it.Updated != want.Updated ||
		it.Size != want.Size || it.URL != want.URL || !it.Adult || it.Endorsements != want.Endorsements || it.Downloads != want.Downloads {
		t.Fatalf("item %+v", it)
	}
}

func TestDetailsResolveAndLatest(t *testing.T) {
	t.Parallel()
	d := fake(t)
	ctx := context.Background()
	det, err := d.Details(ctx, "3659", "42", "")
	if err != nil || det.Description != "Hello **you**" || det.Changelog != "fixed" || len(det.Versions) != 2 || det.Versions[1] != "7" ||
		len(det.Dependencies) != 1 || det.Dependencies[0] != "Harmony" || det.Categories[0] != "Mod" {
		t.Fatalf("%+v %v", det, err)
	}
	cur, err := d.Resolve(ctx, "3659", "42", "")
	if err != nil || cur.FileID != 9 || cur.MD5 != "abc" || cur.URL != "https://g-1.modapi.io/f/9" || cur.Name != "TimberAPI" {
		t.Fatalf("%+v %v", cur, err)
	}
	old, err := d.Resolve(ctx, "3659", "42", "7")
	if err != nil || old.FileID != 7 || old.Version != "7" || old.MD5 != "def" {
		t.Fatalf("%+v %v", old, err)
	}
	if _, err := d.Resolve(ctx, "3659", "42", "nope"); err == nil {
		t.Fatal("unknown file resolved")
	}
	latest, err := d.Latest(ctx, components.GameSource{ID: "modio", Key: "3659"}, "", []source.InstalledFile{
		{ID: "42", Version: "7", Digest: "md5:def"}, {ID: "42", Version: "0.8", Digest: "md5:abc"},
	})
	if err != nil || len(latest) != 1 || latest["md5:def"].VersionID != "9" || latest["md5:def"].Version != "0.8" {
		t.Fatalf("%+v %v", latest, err)
	}
}

func TestTokenAndGameKey(t *testing.T) {
	t.Parallel()
	d := fake(t)
	if name, err := d.Validate(context.Background(), "tok"); err != nil || name != "nomad" {
		t.Fatalf("%q %v", name, err)
	}
	if _, err := d.Validate(context.Background(), "revoked"); !errors.Is(err, ErrBadToken) {
		t.Fatalf("revoked token: %v", err)
	}
	none := Driver{URL: d.URL, Token: func() (string, error) { return "", secret.ErrNotFound }}
	if none.Unavailable() == "" {
		t.Fatal("no token but available")
	}
	if _, err := none.Search(context.Background(), source.Query{Key: "3659"}); !errors.Is(err, ErrNeedsToken) {
		t.Fatalf("search without token: %v", err)
	}
	if _, err := (Driver{Token: d.Token}).Search(context.Background(), source.Query{Key: "evil.host/x"}); err == nil {
		t.Fatal("a non-numeric game key reached the address")
	}
	if d.ModPageURL("3659", "42") != "https://mod.io/g/3659/m/42" || d.ModPageURL("x", "42") != "" {
		t.Fatal("page link")
	}
}
