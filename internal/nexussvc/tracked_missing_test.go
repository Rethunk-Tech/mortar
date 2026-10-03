package nexussvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/zalando/go-keyring"
)

func TestTrackedMissingFiltersDomainAndProfileMods(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
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
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	items, err := modstore.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	profPath := filepath.Join(root, "profiles", "stardew", p.ID, "profile.json")
	b, err := os.ReadFile(profPath)
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
