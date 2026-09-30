package nexussvc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/secret"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

func TestSignInOutKeepsKeyOutOfSettings(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Apikey") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":true}`))
	}))
	defer srv.Close()
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	s := NewService(store, c, &meta.Client{})
	ctx := context.Background()

	if _, err := s.SignIn(ctx, "wrong"); err == nil {
		t.Fatal("bad key accepted")
	}
	if _, err := secret.Get(keyName); !errors.Is(err, secret.ErrNotFound) || s.Account().SignedIn {
		t.Fatalf("rejected key left state behind: %v", err)
	}
	acct, err := s.SignIn(ctx, " secret-key ")
	if err != nil || !acct.SignedIn || acct.Name != "Ada" || !acct.Premium {
		t.Fatalf("sign in = %+v, %v", acct, err)
	}
	if got, _ := secret.Get(keyName); got != "secret-key" {
		t.Fatalf("keyring holds %q", got)
	}
	if acct, err = s.SignOut(); err != nil || acct.SignedIn {
		t.Fatalf("sign out = %+v, %v", acct, err)
	}
	if _, err := secret.Get(keyName); !errors.Is(err, secret.ErrNotFound) {
		t.Fatal("key survived sign out")
	}
}

func TestModNameNeedsASignedInAccount(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store, nexus.New("1"), &meta.Client{})
	if _, err := s.ModName(context.Background(), 1); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("got %v", err)
	}
}

func TestDetailsCachedAndServedStaleWhenSignedOut(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	files := map[string]string{
		"/v1/users/validate.json":                          "validate.json",
		"/v1/games/stardewvalley/mods/541.json":            "mod-541.json",
		"/v1/games/stardewvalley/mods/541/files.json":      "files-541.json",
		"/v1/games/stardewvalley/mods/541/changelogs.json": "changelogs-541.json",
		"/v1/games/stardewvalley.json":                     "game.json",
	}
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		b, err := fsx.ReadFile(filepath.Join("..", "nexus", "testdata", files[r.URL.Path]))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	m := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	s := NewService(store, c, m)
	ctx := context.Background()

	if _, err := s.Details(ctx, 541); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("signed out with no cache = %v", err)
	}
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	hits.Store(0)
	d, err := s.Details(ctx, 541)
	if err != nil || d.Page.Name != "Lookup Anything" || d.Category != "User Interface" || len(d.Files) != 2 ||
		len(d.Changelogs) != 4 || d.Changelogs[0].Version != "1.8.2" {
		t.Fatalf("details = %+v, %v", d, err)
	}
	if hits.Load() != 4 {
		t.Fatalf("hits = %d, want page, files, changelogs and categories", hits.Load())
	}
	if _, err := s.Details(ctx, 541); err != nil || hits.Load() != 4 {
		t.Fatalf("fresh cache refetched: hits = %d, %v", hits.Load(), err)
	}

	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(48 * time.Hour)
	if got := s.CachedDetails([]int{541, 999}); len(got) != 1 || got[541].Category != "User Interface" ||
		hits.Load() != 4 {
		t.Fatalf("cached details = %+v, hits %d", got, hits.Load())
	}
	if d, err := s.Details(ctx, 541); err != nil || d.Page.Name != "Lookup Anything" || hits.Load() != 4 {
		t.Fatalf("stale signed-out details = %+v, %v, hits %d", d.Page, err, hits.Load())
	}
}
