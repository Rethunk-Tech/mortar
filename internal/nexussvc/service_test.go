package nexussvc

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

// signIn is the state a browser sign-in leaves: a grant in the keyring and the account in settings.
func signIn(t *testing.T, s *Service) {
	t.Helper()
	if err := tokenStore.Save(sso.Tokens{Access: "tok", Refresh: "r", Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.signedIn(nexusmods.User{ID: 7, Name: "Ada", IsPremium: true}); err != nil {
		t.Fatal(err)
	}
}

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
		if r.URL.Path == "/v2/graphql" {
			_, _ = w.Write([]byte(`{"data":{"legacyModsByDomain":{"nodes":[{"modId":541,"gameId":1303,"modRequirements":{"nexusRequirements":{"nodes":[{"modId":"2400","modName":"SMAPI","url":"","externalRequirement":false,"notes":"","gameId":"1303"}]}}}]}}}`))
			return
		}
		b, err := safefs.ReadFile(filepath.Join("..", "nexus", "testdata", files[r.URL.Path]))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// keepsOutOfSettings fails when settings.json, which holds the signed-in account's name, also holds a secret.
func keepsOutOfSettings(t *testing.T, secrets ...string) {
	t.Helper()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	b, err := safefs.ReadFile(filepath.Join(dir, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range secrets {
		if strings.Contains(string(b), sec) {
			t.Fatalf("settings.json holds %q: %s", sec, b)
		}
	}
}

func TestModNameNeedsASignedInAccount(t *testing.T) {
	store := testStore(t)
	s := NewService(store, nexusmods.New("Mortar", "1"), &meta.Client{})
	if _, err := s.ModName(t.Context(), "stardew", 1); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("got %v", err)
	}
}

func TestDetailsCachedAndServedStaleWhenSignedOut(t *testing.T) {
	srv, hits := serveFixtures(t)
	store := testStore(t)
	c := nexusmods.New("Mortar", "1")
	c.BaseURL = srv.URL
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	m := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	s := NewService(store, c, m)
	ctx := t.Context()

	if _, err := s.Details(ctx, "stardew", 541); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("signed out with no cache = %v", err)
	}
	signIn(t, s)
	hits.Store(0)
	d, err := s.Details(ctx, "stardew", 541)
	if err != nil || d.Page.Name != "Lookup Anything" || d.Category != "User Interface" || len(d.Files) != 2 ||
		len(d.Changelogs) != 4 || d.Changelogs[0].Version != "1.8.2" {
		t.Fatalf("details = %+v, %v", d, err)
	}
	if r := d.Page.Requirements; len(r) != 1 || r[0].ModID != 2400 || r[0].URL != nexusmods.ModURL("stardewvalley", 2400) {
		t.Fatalf("requirements = %+v", r)
	}
	if hits.Load() != 5 {
		t.Fatalf("hits = %d, want page, requirements, files, changelogs and categories", hits.Load())
	}
	if _, err := s.Details(ctx, "stardew", 541); err != nil || hits.Load() != 5 {
		t.Fatalf("fresh cache refetched: hits = %d, %v", hits.Load(), err)
	}

	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(48 * time.Hour)
	if got := s.CachedDetails("stardew", []int{541, 999}); len(got) != 1 || got[541].Category != "User Interface" ||
		hits.Load() != 5 {
		t.Fatalf("cached details = %+v, hits %d", got, hits.Load())
	}
	_ = diskcache.Put(s.meta.Cache(), PageName("stardewvalley", 777), Details{Page: nexusmods.Page{Version: "3.0"}, Partial: true})
	if got := s.CachedDetails("stardew", []int{777}); got[777].Page.Version != "3.0" || hits.Load() != 5 {
		t.Fatalf("a mod known only from a batched page: %+v, hits %d", got, hits.Load())
	}
	if d, err := s.Details(ctx, "stardew", 541); err != nil || d.Page.Name != "Lookup Anything" || hits.Load() != 5 {
		t.Fatalf("stale signed-out details = %+v, %v, hits %d", d.Page, err, hits.Load())
	}
}

func TestDetailsRefetchesUnversionedCache(t *testing.T) {
	srv, hits := serveFixtures(t)
	store := testStore(t)
	c := nexusmods.New("Mortar", "1")
	c.BaseURL = srv.URL
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	m := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	s := NewService(store, c, m)
	ctx := t.Context()
	signIn(t, s)
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
	testfs.WriteAt(t, path, b)
	hits.Store(0)
	d, err := s.Details(ctx, "stardew", 541)
	if err != nil || d.Changelogs[0].Version != "1.8.2" {
		t.Fatalf("details = %+v, %v", d, err)
	}
	if hits.Load() == 0 {
		t.Fatal("unversioned cache was served")
	}
}

func TestBrowserSignInKeepsTheGrantOutOfSettings(t *testing.T) {
	store := testStore(t)
	api, _ := serveFixtures(t)
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userinfo" {
			_, _ = w.Write([]byte(`{"sub":"77","name":"OAuthName","membership_roles":["member","premium"]}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"A1","refresh_token":"R1","expires_in":3600}`))
	}))
	defer idp.Close()
	client := nexusmods.New("Mortar", "test")
	client.BaseURL = api.URL
	s := NewService(store, client, &meta.Client{})
	s.oauth = sso.OAuth{ClientID: "cid", Store: oauthStore{}, AuthBase: idp.URL, Addr: "127.0.0.1:0", OpenBrowser: func(raw string) error {
		u, _ := url.Parse(raw)
		go func() {
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, u.Query().Get("redirect_uri")+"?code=C&state="+u.Query().Get("state"), nil)
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}}
	if acct, err := s.StartSSO(t.Context()); err != nil || !acct.SignedIn || acct.Name != "OAuthName" || !acct.Premium {
		t.Fatalf("acct %+v err %v", acct, err)
	}
	if tok, err := tokenStore.Load(); err != nil || tok.Refresh != "R1" {
		t.Fatalf("tokens %+v err %v", tok, err)
	}
	keepsOutOfSettings(t, "A1", "R1")
	if _, err := s.SignOut(); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenStore.Load(); err == nil {
		t.Fatal("OAuth tokens survived sign-out")
	}
}

func TestPrimeDetailsAsksOncePerBatch(t *testing.T) {
	var requests atomic.Int32
	var down atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/users/validate.json" {
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
			return
		}
		if down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		requests.Add(1)
		var in struct {
			Variables struct {
				IDs []struct {
					ModID int `json:"modId"`
				} `json:"ids"`
			} `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		var nodes []string
		for _, id := range in.Variables.IDs {
			nodes = append(nodes, `{"modId":`+strconv.Itoa(id.ModID)+`,"name":"Mod","version":"2.0","status":"published","category":"Misc","endorsements":3}`)
		}
		_, _ = w.Write([]byte(`{"data":{"legacyModsByDomain":{"nodes":[` + strings.Join(nodes, ",") + `]}}}`))
	}))
	t.Cleanup(srv.Close)
	store := testStore(t)
	c := nexusmods.New("Mortar", "1")
	c.BaseURL = srv.URL
	s := NewService(store, c, &meta.Client{CacheDir: t.TempDir()})
	ctx := t.Context()
	signIn(t, s)
	ids := make([]int, 800)
	for i := range ids {
		ids[i] = i + 1
	}
	got, err := s.Prime(ctx, "lethal-company", ids)
	if err != nil || len(got) != 800 || !got[5].Partial || got[5].Page.Version != "2.0" || got[5].Category != "Misc" {
		t.Fatalf("got %d, %v, %+v", len(got), err, got[5])
	}
	if int(requests.Load()) != nexusmods.Requests(800) {
		t.Fatalf("800 mods cost %d requests", requests.Load())
	}
	before := requests.Load()
	if _, err := s.Prime(ctx, "lethal-company", ids); err != nil || requests.Load() != before {
		t.Fatalf("a fresh cache must not be asked again: %d requests, %v", requests.Load()-before, err)
	}
	// A grant the keyring cannot hand over (locked, or no keyring running) does not hold back the public page data.
	keyring.MockInitWithError(errors.New("the keyring is locked"))
	if got, err := s.Prime(ctx, "lethal-company", []int{900}); err != nil || got[900].Category != "Misc" {
		t.Fatalf("locked keyring: %+v, %v", got[900], err)
	}
	// Nexus fails for the one new mod: the window still gets the 800 it has, with the reason for the rest.
	down.Store(true)
	primed, err := s.PrimeDetails(ctx, "lethal-company", append(slices.Clone(ids), 801))
	if err != nil || len(primed.Details) != 800 || primed.Error == "" {
		t.Fatalf("a failed fetch: %d details, error %q, %v", len(primed.Details), primed.Error, err)
	}
}

func TestPrimeDetailsDoesNotAskAgainForAModNexusDoesNotReturn(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/users/validate.json" {
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
			return
		}
		requests.Add(1)
		_, _ = w.Write([]byte(`{"data":{"legacyModsByDomain":{"nodes":[]}}}`))
	}))
	t.Cleanup(srv.Close)
	c := nexusmods.New("Mortar", "1")
	c.BaseURL = srv.URL
	s := NewService(testStore(t), c, &meta.Client{CacheDir: t.TempDir()})
	ctx := t.Context()
	signIn(t, s)
	for range 3 {
		if got, err := s.Prime(ctx, "lethal-company", []int{9999}); err != nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("a mod Nexus does not return was asked for %d times", requests.Load())
	}
}

func TestSignedOutCarriesItsKindAndStaysASourceSignedOut(t *testing.T) {
	if usererr.KindOf(ErrSignedOut) != usererr.SignedOut {
		t.Fatalf("kind %v", usererr.KindOf(ErrSignedOut))
	}
	if !errors.Is(ErrSignedOut, source.ErrSignedOut) {
		t.Fatal("no longer a source.ErrSignedOut")
	}
}
