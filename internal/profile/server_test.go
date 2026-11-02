package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

const vsGame = "vintage-story"

func TestServerModeRoundTripsAndOldProfilesAreClients(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create(vsGame, "P")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(vsGame, p.ID)
	raw, err := safefs.ReadFile(filepath.Join(dir, "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, has := doc["server"]; has || e.ServerMode(vsGame, p.ID) {
		t.Fatalf("a profile written before the mode existed must read as a client: %s", raw)
	}
	if _, err := e.SetServerMode(vsGame, p.ID, true); err != nil {
		t.Fatal(err)
	}
	if !e.ServerMode(vsGame, p.ID) {
		t.Fatal("the mode must persist")
	}
	if _, err := e.SetServerMode(vsGame, p.ID, false); err != nil || e.ServerMode(vsGame, p.ID) {
		t.Fatalf("back to the client: %v", err)
	}
}

func TestMakeServerCopiesTheModsAndModConfigIntoAServerProfile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create(vsGame, "Group")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(vsGame, p.ID)
	cfg := filepath.Join(dir, ModConfigFolder, "mod.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	srv, err := e.MakeServer(vsGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !srv.Server || srv.ID == p.ID || srv.Name != "Group server" {
		t.Fatalf("server profile = %+v", srv)
	}
	sdir, _ := e.ProfileDir(vsGame, srv.ID)
	if b, err := safefs.ReadFile(filepath.Join(sdir, ModConfigFolder, "mod.json")); err != nil || string(b) != `{"a":1}` {
		t.Fatalf("mod config not copied: %q %v", b, err)
	}
	if e.ServerMode(vsGame, p.ID) {
		t.Fatal("the source profile became a server")
	}
	if _, err := e.MakeServer("stardew", p.ID); err == nil {
		t.Fatal("a game without a dedicated server made one")
	}
}
