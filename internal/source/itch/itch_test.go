package itch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func fake(t *testing.T) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k1" {
			_, _ = w.Write([]byte(`{"errors":["invalid key"]}`))
			return
		}
		if strings.Contains(r.URL.String(), "k1") {
			t.Errorf("key in address %s", r.URL)
		}
		switch r.URL.Path {
		case "/me":
			_, _ = w.Write([]byte(`{"user":{"username":"farmer"}}`))
		case "/search/games":
			if r.URL.Query().Get("query") != "stardew cool" || r.URL.Query().Get("page") != "2" {
				t.Errorf("params %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"games":[{"id":7,"title":"Cool Mod","short_text":"hi","url":"https://a.itch.io/cool","cover_url":"c.png","published_at":"2026-01-01","user":{"username":"a"}}]}`))
		case "/game/7/uploads":
			_, _ = w.Write([]byte(`{"uploads":[{"id":1,"filename":"demo.zip","demo":true},{"id":2,"filename":"cool.zip","size":9},{"id":3,"filename":"ost.zip","type":"soundtrack"}]}`))
		case "/upload/2/download":
			_, _ = w.Write([]byte(`{"url":"https://cdn/cool.zip"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return Driver{URL: srv.URL, Key: func() (string, error) { return "k1", nil }}
}

func TestSearchAndResolve(t *testing.T) {
	t.Parallel()
	d := fake(t)
	page, err := d.Search(context.Background(), source.Query{Key: "stardew", Text: "cool", Page: 2})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "7" || page.Items[0].Author != "a" || page.Total != 21 {
		t.Fatalf("%+v %v", page, err)
	}
	r, err := d.Resolve(context.Background(), "7", "")
	if err != nil || r.UploadID != "2" || r.URL != "https://cdn/cool.zip" || r.Size != 9 {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := d.Resolve(context.Background(), "7", "nope.zip"); err == nil {
		t.Fatal("unknown upload resolved")
	}
	if d.ModPageURL("", "7") != "https://itch.io/games/7" {
		t.Fatal("link")
	}
}

func TestKeyHandling(t *testing.T) {
	t.Parallel()
	d := fake(t)
	if name, err := d.Validate(context.Background(), " k1 "); err != nil || name != "farmer" {
		t.Fatalf("%q %v", name, err)
	}
	if _, err := d.Validate(context.Background(), "bad"); !errors.Is(err, ErrBadKey) {
		t.Fatalf("bad key: %v", err)
	}
	none := Driver{Key: func() (string, error) { return " ", nil }}
	if none.Unavailable() != "needs an itch.io API key" || d.Unavailable() != "" {
		t.Fatal("unavailable text")
	}
	if _, err := none.Search(context.Background(), source.Query{Key: "x"}); !errors.Is(err, ErrNeedsKey) {
		t.Fatalf("search without key: %v", err)
	}
}
