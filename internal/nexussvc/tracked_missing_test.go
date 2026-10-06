package nexussvc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/testenv"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestTrackedMissingFiltersDomainAndProfileMods(t *testing.T) {
	store := testStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/users/validate.json":
			_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":false}`))
		case "/v1/user/tracked_mods.json":
			_, _ = w.Write([]byte(`[
				{"mod_id":100,"domain_name":"stardewvalley"},
				{"mod_id":200,"domain_name":"stardewvalley"},
				{"mod_id":300,"domain_name":"skyrim"}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := nexus.New("1")
	c.BaseURL = srv.URL
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	profPath := filepath.Join(root, "profiles", "stardew", p.ID, "profile.json")
	b, err := fsx.ReadFile(profPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	raw["entries"] = []any{map[string]any{
		"key":    "nexus-100-1",
		"mods":   []any{},
		"source": map[string]any{"kind": profile.KindNexus, "modId": 100},
	}}
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profPath, out, 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewService(store, c, &meta.Client{})
	s.Profiles = profiles
	ctx := context.Background()
	if _, err := s.SignIn(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	missing, err := s.TrackedMissing(ctx, "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0].ModID != 200 || missing[0].DomainName != "stardewvalley" {
		t.Fatalf("missing = %+v", missing)
	}
}

func TestTrackedMissingIsEmptyWhenTheKeyIsGone(t *testing.T) {
	store := testStore(t)
	if _, err := store.Update(func(s *settings.Settings) { s.NexusUserID = 7 }); err != nil {
		t.Fatal(err)
	}
	s := NewService(store, nexus.New("1"), &meta.Client{})
	if _, err := Authed(store, nexus.New("1")); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("a remembered account without a stored key is signed out: %v", err)
	}
	missing, err := s.TrackedMissing(context.Background(), "stardew", "any")
	if err != nil || len(missing) != 0 {
		t.Fatalf("signed out lists nothing and is not an error: %v %v", missing, err)
	}
}
