package profile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/backup"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

func TestUpdateDeleteOldVersionCarriesConfigOnly(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	mNew := strings.TrimSuffix(m, `, /* c */}`) + `,"DeleteOldVersion":true}`
	e, p := updEnv(t,
		map[string]string{"A/manifest.json": m, "A/tweaked.json": "v1", "A/both.json": "v1"},
		map[string]string{"A/manifest.json": mNew, "A/tweaked.json": "v1", "A/both.json": "v2"})
	writeFile(t, e.mods(p.ID), "a-1/A/config.json", "mine")
	writeFile(t, e.mods(p.ID), "a-1/A/tweaked.json", "mine")
	writeFile(t, e.mods(p.ID), "a-1/A/both.json", "mine")

	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.mods(p.ID), "a-2", "A")
	want := map[string]string{
		"config.json": "mine", "tweaked.json": "v1", "both.json": "v2",
	}
	for rel, body := range want {
		if g := read(t, filepath.Join(dir, rel)); g != body {
			t.Errorf("%s = %q, want %q", rel, g, body)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "both.json.mortar-old")); err == nil {
		t.Error("DeleteOldVersion kept a conflict copy")
	}
}

func TestCarryOverMatrix(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t,
		map[string]string{"Pack/A/manifest.json": m, "Pack/A/cfg.json": "v1"},
		map[string]string{"Pack/Moved/A/manifest.json": m, "Pack/Moved/A/cfg.json": "v2"})
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:me.a", false); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.mods(p.ID), "a-1/Pack/.A/data.json", "save")

	got, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Entries[0].Disabled, []mod.ID{"smapi:me.a"}) || got.Entries[0].Mods[0].Folder != "Pack/Moved/A" {
		t.Fatalf("entry = %+v", got.Entries[0])
	}
	if b := read(t, filepath.Join(e.mods(p.ID), "a-2", "Pack", "Moved", ".A", "cfg.json")); b != "v2" {
		t.Fatalf("cfg = %q", b)
	}

	back, err := e.RollBack("stardew", p.ID, "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if en := back.Entries[0]; en.Key != "a-1" || en.PreviousKey != "a-2" || !slices.Equal(en.Disabled, []mod.ID{"smapi:me.a"}) {
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
	t.Parallel()
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
	t.Parallel()
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

// The game's save folder comes from the process's config folder, so this test points it somewhere private and
// cannot run in parallel.
func TestUpdateBacksUpSavesAndHonoursLock(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	cfg, _ := os.UserConfigDir()
	writeFile(t, filepath.Join(cfg, "StardewValley", "Saves"), "Farm_1/Farm_1", "save")

	backups, _, err := backup.Locations(e.dataDir, "", "stardew")
	if err != nil {
		t.Fatal(err)
	}
	e.Running = func(_, id string) bool { return id == p.ID }
	var re *RunningError
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); !errors.As(err, &re) {
		t.Fatalf("update err = %v", err)
	}
	if _, err := e.RollBack("stardew", p.ID, "a-1"); !errors.As(err, &re) {
		t.Fatalf("rollback err = %v", err)
	}
	if _, err := os.Stat(backups); err == nil {
		t.Fatal("locked update took a backup")
	}

	e.Running = nil
	if _, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2"); err != nil {
		t.Fatal(err)
	}
	got := names(t, backups)
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
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("a read-only folder is made with chmod, which Windows ignores for directories")
	}
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
	t.Parallel()
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
	t.Parallel()
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

func TestUpdateEntriesIsOneChangeAndAtomic(t *testing.T) {
	t.Parallel()
	m := manifestJSON("me.a")
	e, p := updEnv(t, map[string]string{"A/manifest.json": m}, map[string]string{"A/manifest.json": m})
	e.item(t, "b-1", map[string]string{"B/manifest.json": manifestJSON("me.b")})
	e.item(t, "b-2", map[string]string{"B/manifest.json": manifestJSON("me.b")})
	if _, err := e.AddEntry("stardew", p.ID, "b-1", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}

	// The second move names an entry the profile lacks, so the first must not stick.
	bad := []EntryMove{{"a-1", "a-2"}, {"nope", "b-2"}}
	if _, err := e.UpdateEntries("stardew", p.ID, bad); err == nil {
		t.Fatal("moved an entry that is not in the profile")
	}
	if got := names(t, e.mods(p.ID)); !slices.Equal(got, []string{"a-1", "b-1"}) {
		t.Fatalf("mods/ after failure = %v", got)
	}

	// A repeated move (an item with several mods yields one per mod) applies once, as one history event.
	ok := []EntryMove{{"a-1", "a-2"}, {"a-1", "a-2"}, {"b-1", "b-2"}}
	got, err := e.UpdateEntries("stardew", p.ID, ok)
	if err != nil {
		t.Fatal(err)
	}
	if keys := []string{got.Entries[0].Key, got.Entries[1].Key}; !slices.Equal(keys, []string{"a-2", "b-2"}) {
		t.Fatalf("keys = %v", keys)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after)-len(before) != 1 {
		t.Fatalf("history grew by %d events, want 1", len(after)-len(before))
	}
	if _, err := e.UpdateEntries("stardew", p.ID, []EntryMove{{"a-2", "a-1"}, {"a-2", "b-1"}}); err == nil {
		t.Fatal("accepted two targets for one entry")
	}
}

// lcSteamHome is a home with Lethal Company in a Steam library and one save in its Proton prefix; it returns the
// folder backups of it land in.
func lcSteamHome(t *testing.T, e *env) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".local", "share", "Steam")
	e.home = filepath.Dir(filepath.Dir(filepath.Dir(root)))
	apps := filepath.Join(root, "steamapps")
	writeFile(t, apps, "libraryfolders.vdf", "\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \""+root+"\"\n}\n}\n")
	writeFile(t, apps, "appmanifest_1966720.acf", "\"AppState\"\n{\n\"installdir\" \"Lethal Company\"\n}\n")
	writeFile(t, apps, "common/Lethal Company/Lethal Company.exe", "exe")
	writeFile(t, apps, "compatdata/1966720/pfx/drive_c/users/steamuser/AppData/LocalLow/ZeekerssRBLX/Lethal Company/LCSaveFile1", "save")
	backups, _, err := backup.Locations(e.dataDir, "", "lethal-company")
	if err != nil {
		t.Fatal(err)
	}
	return backups
}

func zipsIn(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	for _, name := range names(t, dir) {
		if strings.HasSuffix(name, ".zip") {
			n++
		}
	}
	return n
}

func TestPackageUpdatesBackUpSavesOncePerBatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	backups := lcSteamHome(t, &e)
	a, _ := e.Create("lethal-company", "A")
	b, _ := e.Create("lethal-company", "B")
	v1 := Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"}
	for _, p := range []Profile{a, b} {
		if _, err := e.InstallSource("lethal-company", p.ID, tsZip(t, "1.0.0"), v1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(backups); err == nil {
		t.Fatal("an install took a backup")
	}
	res, err := e.InstallSource("lethal-company", a.ID, tsZip(t, "1.1.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.1.0"})
	if err != nil || !res.Updated {
		t.Fatalf("update = %+v, %v", res, err)
	}
	if n := zipsIn(t, backups); n != 1 {
		t.Fatalf("a package update took %d backups, want 1", n)
	}
	got, err := e.UpdateEverywhere("lethal-company", "thunderstore:Ns-Mod", latestStoreKey)
	if err != nil || len(got.Updated) != 1 || got.Updated[0].ProfileID != b.ID {
		t.Fatalf("everywhere = %+v, %v", got, err)
	}
	newKey := got.Updated[0].OldKey
	pb, _ := e.read("lethal-company", b.ID)
	if en := pb.Entries[0]; en.Source.Version != "1.1.0" || en.PreviousKey != newKey {
		t.Fatalf("b's entry = %+v", en)
	}
	newKey = pb.Entries[0].Key
	rolled, err := e.RollBack("lethal-company", b.ID, newKey)
	if err != nil || rolled.Entries[0].Source.Version != "1.0.0" || rolled.Entries[0].Mods[0].Version != "1.0.0" {
		t.Fatalf("roll back = %+v, %v", rolled.Entries, err)
	}
	if _, err := e.UpdateEntries("lethal-company", b.ID, []EntryMove{{OldKey: rolled.Entries[0].Key, NewKey: newKey}}); err != nil {
		t.Fatal(err)
	}
	if n := zipsIn(t, backups); n != 1 {
		t.Fatalf("the batch took %d backups, want 1", n)
	}
}

// A share names a file from the entry's source, so updating to another store item must move the source with it.
func TestUpdateEntryMovesNexusSourceToTheNewFile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"A/manifest.json": manifestJSON("me.a")}
	e.item(t, "nexus-5-10", files)
	e.item(t, "nexus-5-11", files)
	if _, err := e.AddEntry("stardew", p.ID, "nexus-5-10", Source{Kind: KindNexus, ModID: 5, FileID: 10, Digest: "sha512:old"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.UpdateEntry("stardew", p.ID, "nexus-5-10", "nexus-5-11")
	if err != nil {
		t.Fatal(err)
	}
	src := got.Entries[0].Source
	if got.Entries[0].Key != "nexus-5-11" || src.ModID != 5 || src.FileID != 11 || src.Digest != "" {
		t.Fatalf("entry %q source = %+v, want mod 5 file 11 with no stale digest", got.Entries[0].Key, src)
	}
}
