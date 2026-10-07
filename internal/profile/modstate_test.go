package profile

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestModFolderByKeyPicksTheCopy(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m + " "})
	if _, err := e.AddEntry("stardew", p.ID, "a-2", Source{Kind: KindLocal}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"a-1", "a-2"} {
		got, err := e.ModFolder("stardew", p.ID, key, "smapi:me.a")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(e.mods(p.ID), key, "A"); got != want {
			t.Errorf("ModFolder(%s) = %s, want %s", key, got, want)
		}
	}
	if _, err := e.ModFolder("stardew", p.ID, "a-3", "smapi:me.a"); err == nil {
		t.Error("unknown key accepted")
	}
}

func TestInstalledReturnsModFolders(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m + " "})

	installed, err := e.Installed("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("Installed returned %d mods, want 1", len(installed))
	}
	for _, im := range installed {
		want, err := e.ModFolder("stardew", p.ID, im.Key, im.ModID())
		if err != nil {
			t.Fatal(err)
		}
		if im.Folder != want {
			t.Errorf("Installed(%s).Folder = %s, want %s", im.Key, im.Folder, want)
		}
	}
}

func TestModStateAndResetConfig(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t,
		map[string]string{"A/manifest.json": m, "A/config.json": "shipped"},
		map[string]string{"A/manifest.json": m + " "})
	svc := NewService(e.Store, t.TempDir(), nil)
	state := func() ModState {
		t.Helper()
		st, err := svc.ModState("stardew", p.ID, "a-1", "smapi:me.a")
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
	if err := svc.ResetConfig("stardew", p.ID, "a-1", "smapi:me.a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatalf("config.json survived a reset: %v", err)
	}
	if st := state(); st.Config != ConfigNone {
		t.Fatalf("after reset = %+v", st)
	}
	if err := svc.ResetConfig("stardew", p.ID, "a-1", "smapi:me.a"); err != nil {
		t.Fatalf("second reset: %v", err)
	}
}

func TestConfigPathStaysInTheModFolder(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m + " "})
	if _, err := e.ConfigPath("stardew", p.ID, "a-1", "smapi:me.a"); err == nil {
		t.Fatal("missing config.json opened")
	}
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	got, err := e.ConfigPath("stardew", p.ID, "a-1", "smapi:me.a")
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
	if err := svc.OpenConfig("stardew", p.ID, "a-1", "smapi:nope.Mod"); err == nil {
		t.Fatal("unknown mod opened")
	}
}

func TestReadWriteConfigRoundTripAndLock(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/config.json": `{"z":1,"n":1.5}`}, map[string]string{"A/manifest.json": m + " "})
	svc := NewService(e.Store, t.TempDir(), nil)
	got, err := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.a")
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\n  \"z\": 1,\n  \"n\": 1.5\n}\n" {
		t.Fatalf("read = %s", got)
	}
	if err := svc.WriteConfig("stardew", p.ID, "a-1", "smapi:me.a", `{"z":1,"n":1.5,"s":"ok"}`); err != nil {
		t.Fatal(err)
	}
	got, err = svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.a")
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "z": 1,
  "n": 1.5,
  "s": "ok"
}
`
	if got != want {
		t.Fatalf("wrote:\n%s", got)
	}
	e.Running = func(_, id string) bool { return id == p.ID }
	if err := svc.WriteConfig("stardew", p.ID, "a-1", "smapi:me.a", `{"z":2}`); err == nil {
		t.Fatal("write while running")
	} else if _, ok := errors.AsType[*RunningError](err); !ok {
		t.Fatalf("err = %v, want RunningError", err)
	}
	if _, err := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:nope.Mod"); err == nil {
		t.Fatal("unknown mod read")
	}
	if err := svc.WriteConfig("stardew", p.ID, "a-1", "smapi:nope.Mod", `{}`); err == nil {
		t.Fatal("unknown mod write")
	}
}

func TestReadConfigKeepsLargeNumbersAndComments(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/config.json": "{\n// accepted\n\"large\": 9007199254740993,\n\"tail\": [1,],\n}"}, nil)
	svc := NewService(e.Store, t.TempDir(), nil)
	got, err := svc.ReadConfig("stardew", p.ID, "a-1", "smapi:me.a")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "9007199254740993") {
		t.Fatalf("large number changed: %s", got)
	}
	if strings.Contains(got, "//") || strings.Contains(got, ",]") {
		t.Fatalf("JSON noise remained: %s", got)
	}
}

func TestRollBackThroughService(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	v2 := `{"Name":"me.a","Author":"me","Version":"2.0.0","UniqueID":"me.a"}`
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": v2})
	svc := NewService(e.Store, t.TempDir(), nil)
	if _, err := svc.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	st, err := svc.ModState("stardew", p.ID, "a-2", "smapi:me.a")
	if err != nil || st.PreviousVersion != "1.0.0" {
		t.Fatalf("state = %+v, %v", st, err)
	}
	got, err := svc.RollBack("stardew", p.ID, "a-2")
	if err != nil || got.Entries[0].Key != "a-1" || got.Entries[0].PreviousKey != "a-2" {
		t.Fatalf("rolled back = %+v, %v", got.Entries, err)
	}
}

func TestConfigHistoryNamesTheModNotItsID(t *testing.T) {
	t.Parallel()
	m := `{"Name":"Alpha Mod","Author":"me","Version":"1.0.0","UniqueID":"me.a", /* c */}`
	e, p := updEnv(t, map[string]string{"A/manifest.json": m, "A/config.json": `{"z":1}`}, map[string]string{"A/manifest.json": m + " "})
	svc := NewService(e.Store, t.TempDir(), nil)
	if err := svc.WriteConfig("stardew", p.ID, "a-1", "smapi:me.a", `{"z":2}`); err != nil {
		t.Fatal(err)
	}
	cur, err := e.Store.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var want string
	for _, en := range cur.Entries {
		for _, c := range en.Mods {
			if c.ID.Local() == "me.a" {
				want = c.Name
			}
		}
	}
	events, err := svc.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(events, func(ev HistoryEvent) bool { return ev.Change == ChangeConfigEdited })
	if want == "" || want == "me.a" || i < 0 || events[i].Name != want {
		t.Fatalf("name = %q, want %q (events %+v)", events[i].Name, want, events)
	}
	if got := modDisplayName(cur.Entries, "gone.Mod"); got != "gone.Mod" {
		t.Fatalf("a mod that is gone reads %q", got)
	}
}
