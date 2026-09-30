package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModFolderByKeyPicksTheCopy(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m + " "})
	if _, err := e.AddEntry("stardew", p.ID, "a-2", Source{Kind: KindLocal}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"a-1", "a-2"} {
		got, err := e.ModFolder("stardew", p.ID, key, "me.a")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(e.mods(p.ID), key, "A"); got != want {
			t.Errorf("ModFolder(%s) = %s, want %s", key, got, want)
		}
	}
	if _, err := e.ModFolder("stardew", p.ID, "a-3", "me.a"); err == nil {
		t.Error("unknown key accepted")
	}
}

func TestModStateAndResetConfig(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t,
		map[string]string{"A/manifest.json": m, "A/config.json": "shipped"},
		map[string]string{"A/manifest.json": m + " "})
	svc := NewService(e.Store, t.TempDir(), nil)
	state := func() ModState {
		t.Helper()
		st, err := svc.ModState("stardew", p.ID, "a-1", "me.a")
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	if st := state(); st.Config != ConfigDefault || st.PreviousVersion != "" {
		t.Fatalf("fresh = %+v", st)
	}
	cfg := filepath.Join(e.mods(p.ID), "a-1", "A", "config.json")
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	if st := state(); st.Config != ConfigChanged {
		t.Fatalf("edited = %+v", st)
	}
	if err := svc.ResetConfig("stardew", p.ID, "a-1", "me.a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatalf("config.json survived a reset: %v", err)
	}
	if st := state(); st.Config != ConfigNone {
		t.Fatalf("after reset = %+v", st)
	}
	if err := svc.ResetConfig("stardew", p.ID, "a-1", "me.a"); err != nil {
		t.Fatalf("second reset: %v", err)
	}
}

func TestConfigPathStaysInTheModFolder(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m + " "})
	if _, err := e.ConfigPath("stardew", p.ID, "a-1", "me.a"); err == nil {
		t.Fatal("missing config.json opened")
	}
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	got, err := e.ConfigPath("stardew", p.ID, "a-1", "me.a")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(e.mods(p.ID), "a-1", "A", "config.json")
	if got != want {
		t.Fatalf("ConfigPath = %s, want %s", got, want)
	}
	if rel, err := filepath.Rel(filepath.Join(e.mods(p.ID), "a-1", "A"), got); err != nil || rel != "config.json" {
		t.Fatalf("escaped the mod folder: %s", got)
	}
	svc := NewService(e.Store, t.TempDir(), nil)
	if err := svc.OpenConfig("stardew", p.ID, "a-1", "nope.Mod"); err == nil {
		t.Fatal("unknown mod opened")
	}
}

func TestRollBackThroughService(t *testing.T) {
	m := manifestJSON("me.a")
	v2 := `{"Name":"me.a","Author":"me","Version":"2.0.0","UniqueID":"me.a"}`
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": v2})
	svc := NewService(e.Store, t.TempDir(), nil)
	if _, err := svc.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	st, err := svc.ModState("stardew", p.ID, "a-2", "me.a")
	if err != nil || st.PreviousVersion != "1.0.0" {
		t.Fatalf("state = %+v, %v", st, err)
	}
	got, err := svc.RollBack("stardew", p.ID, "a-2")
	if err != nil || got.Entries[0].Key != "a-1" || got.Entries[0].PreviousKey != "a-2" {
		t.Fatalf("rolled back = %+v, %v", got.Entries, err)
	}
}
