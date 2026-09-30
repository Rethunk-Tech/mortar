package nexus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

var t0 = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fsx.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// serve replays fixtures by path; status overrides every answer when nonzero, and hourly sets the remaining-calls header.
func serve(t *testing.T, status *atomic.Int32, hourly string, hits *atomic.Int32) *Client {
	t.Helper()
	files := map[string]string{
		"/v1/users/validate.json":                                        "validate.json",
		"/v1/games/stardewvalley/mods/541.json":                          "mod-541.json",
		"/v1/games/stardewvalley/mods/541/files.json":                    "files-541.json",
		"/v1/games/stardewvalley/mods/541/changelogs.json":               "changelogs-541.json",
		"/v1/games/stardewvalley.json":                                   "game.json",
		"/v1/games/stardewvalley/mods/541/files/3001/download_link.json": "download-link.json",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Apikey") != "k" || r.Header.Get("Application-Name") != "Mortar" ||
			r.Header.Get("Application-Version") != "9.9" || r.Header.Get("User-Agent") != "Mortar/9.9" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Rl-Daily-Remaining", "19000")
		w.Header().Set("X-Rl-Daily-Limit", "20000")
		w.Header().Set("X-Rl-Daily-Reset", "2026-10-01 00:00:00 +0000")
		w.Header().Set("X-Rl-Hourly-Remaining", hourly)
		w.Header().Set("X-Rl-Hourly-Limit", "2000")
		w.Header().Set("X-Rl-Hourly-Reset", "2026-09-30T13:00:00+00:00")
		if s := status.Load(); s != 0 {
			w.WriteHeader(int(s))
			return
		}
		name, ok := files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(fixture(t, name))
	}))
	t.Cleanup(srv.Close)
	c := New("9.9")
	c.BaseURL, c.CacheDir, c.Now = srv.URL, t.TempDir(), func() time.Time { return t0 }
	return c.WithKey("k")
}

func TestReplays(t *testing.T) {
	var status, hits atomic.Int32
	c := serve(t, &status, "1900", &hits)
	ctx := context.Background()

	u, err := c.Validate(ctx)
	if err != nil || u != (User{ID: 1234567, Name: "TestAccount"}) {
		t.Fatalf("validate = %+v, %v", u, err)
	}
	l := c.Limits()
	if !l.Known || l.Daily.Remaining != 19000 || l.Hourly.Limit != 2000 || !l.Hourly.Reset.Equal(time.Date(2026, 9, 30, 13, 0, 0, 0, time.UTC)) || !l.Daily.Reset.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("limits = %+v", l)
	}

	files, err := c.Files(ctx, 541)
	if err != nil || len(files) != 2 || files[0] != (File{FileID: 3001, FileName: "Content Patcher-541-2-0-0.zip", Name: "Content Patcher", Version: "2.0.0", ModVersion: "2.0.0", Category: "MAIN", SizeKB: 2048, IsPrimary: true, Uploaded: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}) || files[1].Category != "" {
		t.Fatalf("files = %+v, %v", files, err)
	}

	links, err := c.DownloadLinks(ctx, 541, 3001, "", 0)
	if err != nil || len(links) != 1 || links[0].Name != "Nexus CDN" || links[0].URI == "" {
		t.Fatalf("links = %+v, %v", links, err)
	}
}

func TestModCachedOnDisk(t *testing.T) {
	var status, hits atomic.Int32
	c := serve(t, &status, "1900", &hits)
	want := Mod{Name: "Lookup Anything", Author: "Pathoschild", PictureURL: "https://staticdelivery.nexusmods.com/mods/1303/images/541-0-1482010183.png", EndorsementCount: 201516, Summary: "See live info about whatever's under your cursor when you press F1. Learn a villager's favourite gifts, when a crop will be ready to harvest, how long a fence will last, why your farm animals are unhappy, and more."}
	for range 2 {
		m, err := c.Mod(context.Background(), 541)
		if err != nil || m != want {
			t.Fatalf("mod = %+v, %v", m, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want the second call served from disk", hits.Load())
	}
}

func TestPageChangelogsCategories(t *testing.T) {
	var status, hits atomic.Int32
	c := serve(t, &status, "1900", &hits)
	ctx := context.Background()

	p, err := c.Page(ctx, 541)
	if err != nil || p.ModID != 541 || p.Version != "1.55.0" || p.UploadedBy != "Pathoschild" || p.CategoryID != 10 ||
		p.Downloads != 9915155 || p.UniqueDownloads != 3726255 || p.Adult || !p.Available || p.Status != "published" ||
		p.Endorsement != "Endorsed" ||
		!p.Created.Equal(time.Date(2016, 9, 19, 2, 37, 8, 0, time.UTC)) || !p.Updated.Equal(time.Date(2026, 3, 15, 2, 54, 41, 0, time.UTC)) ||
		!strings.HasPrefix(p.Description, "See live info") {
		t.Fatalf("page = %+v, %v", p, err)
	}

	logs, err := c.Changelogs(ctx, 541, 3)
	want := []Changelog{
		{Version: "1.8.2", Notes: []string{"Fixed race condition when rendering ranges of items like sprinklers"}},
		{Version: "1.8.0", Notes: []string{"Updated for Stardew Valley 1.4 and SMAPI 3.0", "Updated Portuguese"}},
		{Version: "1.0.11", Notes: []string{"Fixed custom menu not resizing when zoom options changed.", "Fixed Sebastian's room and the Saloon not existing as map locations."}},
	}
	if err != nil || !reflect.DeepEqual(logs, want) {
		t.Fatalf("changelogs = %+v, %v", logs, err)
	}
	if none, err := parseChangelogs([]byte("[]"), 5); err != nil || len(none) != 0 {
		t.Fatalf("empty changelogs = %+v, %v", none, err)
	}

	cats, err := c.Categories(ctx)
	if err != nil || cats[10] != "User Interface" || cats[1] != "Stardew Valley" {
		t.Fatalf("categories = %v, %v", cats, err)
	}
}

func TestErrors(t *testing.T) {
	var status, hits atomic.Int32
	c := serve(t, &status, "1900", &hits)
	ctx := context.Background()

	status.Store(http.StatusUnauthorized)
	if _, err := c.Validate(ctx); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("401 = %v", err)
	}
	status.Store(http.StatusForbidden)
	if _, err := c.DownloadLinks(ctx, 541, 3001, "", 0); !errors.Is(err, ErrPremiumRequired) {
		t.Fatalf("403 = %v", err)
	}
	var se *StatusError
	if _, err := c.Files(ctx, 541); !errors.As(err, &se) || se.Code != http.StatusForbidden {
		t.Fatalf("403 on files = %v", err)
	}
	status.Store(http.StatusTooManyRequests)
	var re *RateLimitError
	if _, err := c.Validate(ctx); !errors.As(err, &re) || !re.Reset.Equal(time.Date(2026, 9, 30, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("429 = %v", err)
	}
}

func TestRefusesAtFloor(t *testing.T) {
	var status, hits atomic.Int32
	c := serve(t, &status, "3", &hits)
	if _, err := c.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := hits.Load()
	var re *RateLimitError
	if _, err := c.Files(context.Background(), 541); !errors.As(err, &re) || !re.Reset.Equal(time.Date(2026, 9, 30, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("at floor = %v", err)
	}
	if hits.Load() != before {
		t.Fatal("a request was sent at the floor")
	}
	c.Now = func() time.Time { return t0.Add(2 * time.Hour) }
	if _, err := c.Files(context.Background(), 541); err != nil {
		t.Fatalf("after reset = %v", err)
	}
}

func TestChangelogsAreNewestFirstWhateverTheKeyOrder(t *testing.T) {
	raw := []byte(`{"3.5.2":["c"],"1.0":["a"],"not a version":["x"],"2.2.1":["b"],"1.44":["d"]}`)
	got, err := parseChangelogs(raw, 4)
	if err != nil {
		t.Fatal(err)
	}
	var versions []string
	for _, c := range got {
		versions = append(versions, c.Version)
	}
	if want := []string{"3.5.2", "2.2.1", "1.44", "1.0"}; !slices.Equal(versions, want) {
		t.Fatalf("got %v, want %v", versions, want)
	}
}

func TestLimitsHookCanReadTheBudget(t *testing.T) {
	c := New("test")
	got := make(chan Limits, 1)
	c.SetLimitsHook(func(Limits) { got <- c.Limits() })
	h := http.Header{}
	h.Set("X-Rl-Daily-Remaining", "19000")
	h.Set("X-Rl-Hourly-Remaining", "400")
	done := make(chan struct{})
	go func() {
		c.record(h)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("record deadlocked calling a hook that reads Limits")
	}
	if l := <-got; !l.Known || l.Daily.Remaining != 19000 {
		t.Fatalf("hook read %+v", l)
	}
}
