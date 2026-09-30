package profile

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// updEnv is an env with a profile holding entry "a-1" of mod me.a, and store items a-1 and a-2.
func updEnv(t *testing.T, v1, v2 map[string]string) (env, Profile) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	e := newEnv(t)
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	e.item(t, "a-1", v1)
	e.item(t, "a-2", v2)
	if _, err := e.AddEntry("stardew", p.ID, "a-1", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	return e, p
}

func TestCarryOverMatrix(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t,
		map[string]string{"A/manifest.json": m, "A/tweaked.json": "v1", "A/both.json": "v1", "A/same.json": "v1", "A/target.json": "v1"},
		map[string]string{"A/manifest.json": m, "A/tweaked.json": "v1", "A/both.json": "v2", "A/same.json": "v1", "A/target.json": "v2", "A/new.json": "n"})
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	writeFile(t, e.mods(p.ID), "a-1/A/tweaked.json", "mine")
	writeFile(t, e.mods(p.ID), "a-1/A/both.json", "mine")

	got, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if en := got.Entries[0]; en.Key != "a-2" || en.PreviousKey != "a-1" || len(en.Mods) != 1 {
		t.Fatalf("entry = %+v", en)
	}
	dir := filepath.Join(e.mods(p.ID), "a-2", "A")
	want := map[string]string{
		"config.json": "mine", "tweaked.json": "mine", "both.json": "v2", "both.json.mortar-old": "mine",
		"same.json": "v1", "target.json": "v2", "new.json": "n",
	}
	for rel, body := range want {
		if g := read(t, filepath.Join(dir, rel)); g != body {
			t.Errorf("%s = %q, want %q", rel, g, body)
		}
	}
	if got := names(t, e.mods(p.ID)); !slices.Equal(got, []string{"a-2"}) {
		t.Fatalf("mods/ = %v", got)
	}
}

func TestUpdateKeepsDisabledFolderAndRollsBack(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t,
		map[string]string{"Pack/A/manifest.json": m, "Pack/A/cfg.json": "v1"},
		map[string]string{"Pack/Moved/A/manifest.json": m, "Pack/Moved/A/cfg.json": "v2"})
	if _, err := e.SetModEnabled("stardew", p.ID, "", "me.a", false); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.mods(p.ID), "a-1/Pack/.A/data.json", "save")

	got, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Entries[0].Disabled, []string{"me.a"}) || got.Entries[0].Mods[0].Folder != "Pack/Moved/A" {
		t.Fatalf("entry = %+v", got.Entries[0])
	}
	if b := read(t, filepath.Join(e.mods(p.ID), "a-2", "Pack", "Moved", ".A", "cfg.json")); b != "v2" {
		t.Fatalf("cfg = %q", b)
	}

	back, err := e.RollBack("stardew", p.ID, "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if en := back.Entries[0]; en.Key != "a-1" || en.PreviousKey != "a-2" || !slices.Equal(en.Disabled, []string{"me.a"}) {
		t.Fatalf("rolled back = %+v", en)
	}
	if b := read(t, filepath.Join(e.mods(p.ID), "a-1", "Pack", ".A", "cfg.json")); b != "v1" {
		t.Fatalf("cfg = %q", b)
	}
	if _, err := e.RollBack("stardew", p.ID, "a-1"); err != nil {
		t.Fatalf("second roll back: %v", err)
	}
	if _, err := e.RollBack("stardew", p.ID, "a-2"); err != nil {
		t.Fatal(err)
	}
	e.item(t, "b-1", map[string]string{"B/manifest.json": manifestJSON("me.b")})
	if _, err := e.AddEntry("stardew", p.ID, "b-1", Source{}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RollBack("stardew", p.ID, "b-1"); err == nil || !strings.Contains(err.Error(), "no previous version") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallArchiveUpdatesHeldEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	v1 := buildZip(t, "A1.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/cfg.json": "v1"})
	v2 := buildZip(t, "A2.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/cfg.json": "v2"})
	if _, err := e.InstallArchive("stardew", p.ID, v1); err != nil {
		t.Fatal(err)
	}
	first := e.mods(p.ID)

	res, err := e.InstallArchive("stardew", p.ID, v2)
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries
	if !res.Updated || len(en) != 1 || en[0].PreviousKey == "" || en[0].Source.Name != "A2.zip" || len(names(t, first)) != 1 {
		t.Fatalf("result = %+v", res)
	}

	_, err = e.InstallArchive("stardew", p.ID, v2)
	var ie *InstallError
	if !errors.As(err, &ie) || !errors.As(err, new(*DuplicateError)) || !strings.Contains(ie.Msg, "already in this profile") {
		t.Fatalf("same key err = %v", err)
	}
}

func TestInstallArchiveSpanningEntriesFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	for _, id := range []string{"X.A", "X.B"} {
		z := buildZip(t, id+".zip", map[string]string{id + "/manifest.json": manifestJSON(id)})
		if _, err := e.InstallArchive("stardew", p.ID, z); err != nil {
			t.Fatal(err)
		}
	}
	both := buildZip(t, "Both.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "B/manifest.json": manifestJSON("X.B")})
	_, err := e.InstallArchive("stardew", p.ID, both)
	var ie *InstallError
	if !errors.As(err, &ie) || !errors.As(err, new(*SpansEntriesError)) || !strings.Contains(ie.Msg, "X.A; X.B") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateBacksUpSavesAndHonoursLock(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	cfg, _ := os.UserConfigDir()
	writeFile(t, filepath.Join(cfg, "StardewValley", "Saves"), "Farm_1/Farm_1", "save")

	e.Running = func(_, id string) bool { return id == p.ID }
	var re *RunningError
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); !errors.As(err, &re) {
		t.Fatalf("update err = %v", err)
	}
	if _, err := e.RollBack("stardew", p.ID, "a-1"); !errors.As(err, &re) {
		t.Fatalf("rollback err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.base, "backups")); err == nil {
		t.Fatal("locked update took a backup")
	}

	e.Running = nil
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	got := names(t, filepath.Join(e.base, "backups"))
	zips := 0
	for _, n := range got {
		if strings.HasSuffix(n, ".zip") {
			zips++
		}
	}
	if zips != 1 {
		t.Fatalf("backups = %v", got)
	}
}

func TestUpdateThatCannotRecordRestoresTheOldFolder(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	dir := filepath.Dir(e.mods(p.ID))
	if err := fsx.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	_, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2")
	if cerr := fsx.Chmod(dir, 0o700); cerr != nil {
		t.Fatal(cerr)
	}
	if err == nil {
		t.Fatal("update recorded into a read-only profile folder")
	}
	if got := names(t, e.mods(p.ID)); !slices.Equal(got, []string{"a-1"}) {
		t.Fatalf("mods/ = %v", got)
	}
	if g := read(t, filepath.Join(e.mods(p.ID), "a-1", "A", "config.json")); g != "mine" {
		t.Fatalf("config = %q", g)
	}
}

func TestRebuildUndoesAnInterruptedUpdate(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	mods := e.mods(p.ID)
	if err := os.Rename(filepath.Join(mods, "a-1"), filepath.Join(mods, asidePrefix+"a-1")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, mods, "a-2/A/manifest.json", m)
	writeFile(t, mods, ".a-3/A/manifest.json", m)
	if _, err := e.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if got := names(t, mods); !slices.Equal(got, []string{"a-1"}) {
		t.Fatalf("mods/ = %v", got)
	}
	if g := read(t, filepath.Join(mods, "a-1", "A", "config.json")); g != "mine" {
		t.Fatalf("config = %q", g)
	}
}

func TestUpdateKeepsEntrySettingsAndRollBackRestoresSource(t *testing.T) {
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	p, err := e.SetEntryNoteTags("stardew", p.ID, "a-1", "keep me", []string{"ui"})
	if err != nil {
		t.Fatal(err)
	}
	first := p.Entries[0].Source
	replacing := Source{Kind: KindLocal, Name: "a-newer.zip"}
	got, err := e.moveTo("stardew", p.ID, "a-1", "a-2", &replacing)
	if err != nil {
		t.Fatal(err)
	}
	en := got.Entries[0]
	if en.Note != "keep me" || !slices.Equal(en.Tags, []string{"ui"}) || en.Source != replacing {
		t.Fatalf("updated = %+v", en)
	}
	back, err := e.RollBack("stardew", p.ID, "a-2")
	if err != nil {
		t.Fatal(err)
	}
	en = back.Entries[0]
	if en.Key != "a-1" || en.Source != first || en.Note != "keep me" || en.PreviousSource == nil || *en.PreviousSource != replacing {
		t.Fatalf("rolled back = %+v", en)
	}
}
