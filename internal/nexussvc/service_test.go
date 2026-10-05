package nexussvc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussso"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/coder/websocket"
	"github.com/zalando/go-keyring"
)

func testStore(t *testing.T) *settings.Store {
	t.Helper()
	keyring.MockInit()
	testfs.DataHome(t)
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// serveFixtures serves the recorded Nexus responses for mod 541 and counts every request.
func serveFixtures(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
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
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestSignInOutKeepsKeyOutOfSettings(t *testing.T) {
	store := testStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Apikey") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":true}`))
	}))
	defer srv.Close()
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
	store := testStore(t)
	s := NewService(store, nexus.New("1"), &meta.Client{})
	if _, err := s.ModName(context.Background(), "stardew", 1); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("got %v", err)
	}
}

func TestDetailsCachedAndServedStaleWhenSignedOut(t *testing.T) {
	srv, hits := serveFixtures(t)
	store := testStore(t)
	c := nexus.New("1")
	c.BaseURL = srv.URL
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	m := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	s := NewService(store, c, m)
	ctx := context.Background()

	if _, err := s.Details(ctx, "stardew", 541); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("signed out with no cache = %v", err)
	}
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	hits.Store(0)
	d, err := s.Details(ctx, "stardew", 541)
	if err != nil || d.Page.Name != "Lookup Anything" || d.Category != "User Interface" || len(d.Files) != 2 ||
		len(d.Changelogs) != 4 || d.Changelogs[0].Version != "1.8.2" {
		t.Fatalf("details = %+v, %v", d, err)
	}
	if hits.Load() != 4 {
		t.Fatalf("hits = %d, want page, files, changelogs and categories", hits.Load())
	}
	if _, err := s.Details(ctx, "stardew", 541); err != nil || hits.Load() != 4 {
		t.Fatalf("fresh cache refetched: hits = %d, %v", hits.Load(), err)
	}

	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(48 * time.Hour)
	if got := s.CachedDetails("stardew", []int{541, 999}); len(got) != 1 || got[541].Category != "User Interface" ||
		hits.Load() != 4 {
		t.Fatalf("cached details = %+v, hits %d", got, hits.Load())
	}
	if d, err := s.Details(ctx, "stardew", 541); err != nil || d.Page.Name != "Lookup Anything" || hits.Load() != 4 {
		t.Fatalf("stale signed-out details = %+v, %v, hits %d", d.Page, err, hits.Load())
	}
}

func TestDetailsRefetchesUnversionedCache(t *testing.T) {
	srv, hits := serveFixtures(t)
	store := testStore(t)
	c := nexus.New("1")
	c.BaseURL = srv.URL
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	m := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	s := NewService(store, c, m)
	ctx := context.Background()
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	old := struct {
		Fetched time.Time `json:"fetched"`
		Value   Details   `json:"value"`
	}{
		Fetched: now,
		Value:   Details{Changelogs: []nexus.Changelog{{Version: "0.1.0"}}},
	}
	b, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.CacheDir, "nexus", "details-stardewvalley-541.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	hits.Store(0)
	d, err := s.Details(ctx, "stardew", 541)
	if err != nil || d.Changelogs[0].Version != "1.8.2" {
		t.Fatalf("details = %+v, %v", d, err)
	}
	if hits.Load() == 0 {
		t.Fatal("unversioned cache was served")
	}
}

func TestStartSSOStoresKeyLikePasted(t *testing.T) {
	store := testStore(t)
	api, _ := serveFixtures(t)
	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		_, _, _ = c.Read(r.Context())
		_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"success":true,"data":{"connection_token":"t"}}`))
		_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"success":true,"data":{"api_key":"SSOKEY"}}`))
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))
	defer ws.Close()
	client := nexus.New("test")
	client.BaseURL = api.URL
	s := NewService(store, client, &meta.Client{})
	if s.SSOAvailable() {
		t.Fatal("SSO available without a slug")
	}
	if _, err := s.StartSSO(context.Background()); err == nil {
		t.Fatal("StartSSO ran without a slug")
	}
	s.sso = nexussso.Legacy{Slug: "mortar", SocketURL: "ws" + strings.TrimPrefix(ws.URL, "http"), OpenBrowser: func(string) error { return nil }}
	acct, err := s.StartSSO(context.Background())
	if err != nil || !acct.SignedIn {
		t.Fatalf("acct %+v err %v", acct, err)
	}
	if k, _ := secret.Get(keyName); k != "SSOKEY" {
		t.Fatalf("stored key %q", k)
	}
}

func TestStartSSOPrefersOAuthWhenClientIDSet(t *testing.T) {
	store := testStore(t)
	api, _ := serveFixtures(t)
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"A1","refresh_token":"R1","expires_in":3600}`))
	}))
	defer idp.Close()
	client := nexus.New("test")
	client.BaseURL = api.URL
	s := NewService(store, client, &meta.Client{})
	s.sso = nexussso.Legacy{Slug: "mortar"}
	s.oauth = nexussso.OAuth{ClientID: "cid", AuthBase: idp.URL, OpenBrowser: func(raw string) error {
		u, _ := url.Parse(raw)
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, u.Query().Get("redirect_uri")+"?code=C&state="+u.Query().Get("state"), nil)
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}}
	if !s.SSOAvailable() {
		t.Fatal("SSO unavailable with a client id")
	}
	if acct, err := s.StartSSO(context.Background()); err != nil || !acct.SignedIn {
		t.Fatalf("acct %+v err %v", acct, err)
	}
	if tok, err := nexussso.Load(); err != nil || tok.Refresh != "R1" {
		t.Fatalf("tokens %+v err %v", tok, err)
	}
	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	if _, err := nexussso.Load(); err == nil {
		t.Fatal("OAuth tokens survived sign-out")
	}
	s.oauth = nexussso.OAuth{}
	s.sso = nexussso.Legacy{}
	if s.SSOAvailable() {
		t.Fatal("SSO available with neither flag")
	}
}
