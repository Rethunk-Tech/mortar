package profile

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

type env struct {
	*Store
	items *store.Store
	base  string
}

func newEnv(t *testing.T) env {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("LOCALAPPDATA", base)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	return env{&Store{root: filepath.Join(base, "profiles"), trash: filepath.Join(base, "trash"), items: items}, items, base}
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	testfs.WriteFile(t, root, rel, body)
}

// item puts files into the store under key.
func (e env) item(t *testing.T, key string, files map[string]string) {
	t.Helper()
	src := t.TempDir()
	for rel, body := range files {
		writeFile(t, src, rel, body)
	}
	if err := e.items.AddDir("stardew", key, src); err != nil {
		t.Fatal(err)
	}
}

func manifestJSON(id string) string {
	return `{"Name":"` + id + `","Author":"me","Version":"1.0.0","UniqueID":"` + id + `", /* c */}`
}

func (e env) mods(id string) string { return filepath.Join(e.root, "stardew", id, "mods") }

func names(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range ents {
		out = append(out, d.Name())
	}
	return out
}

func TestAddEntryScansAndRejectsDuplicates(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{
		"Pack/A/manifest.json":    manifestJSON("X.A"),
		"Pack/B/manifest.json":    manifestJSON("X.B"),
		"Pack/.off/manifest.json": manifestJSON("X.Off"),
	})
	p, _ := e.Create("stardew", "P")
	got, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	en := got.Entries[0]
	if en.Source.Name != "a.zip" || len(en.Mods) != 2 || en.Mods[0].Folder != "Pack/A" || en.Mods[1].ID != "smapi:X.B" {
		t.Fatalf("entry = %+v", en)
	}
	if en.Added.IsZero() || time.Since(en.Added) > time.Minute {
		t.Fatalf("added = %v", en.Added)
	}
	if en.Mods[0].Needs == nil || en.Mods[1].Needs == nil {
		t.Fatalf("needs = %+v", en.Mods)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), "local-a", "Pack", "A", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	_, err = e.AddEntry("stardew", p.ID, "local-a", Source{})
	if err == nil || !strings.Contains(err.Error(), "X.A, X.B") {
		t.Fatalf("duplicate err = %v", err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-missing", Source{}); err == nil {
		t.Fatal("missing store item accepted")
	}
	e.item(t, "local-empty", map[string]string{"readme.txt": "x"})
	if _, err := e.AddEntry("stardew", p.ID, "local-empty", Source{}); err == nil {
		t.Fatal("entry without a manifest accepted")
	}
	if slices.Contains(names(t, e.mods(p.ID)), "local-empty") || len(names(t, e.mods(p.ID))) != 1 {
		t.Fatalf("mods = %v", names(t, e.mods(p.ID)))
	}
	if _, err := e.AddEntry("stardew", "../x", "local-a", Source{}); err == nil {
		t.Fatal("bad id accepted")
	}
	if _, err := e.AddEntry("stardew", p.ID, "../x", Source{}); err == nil {
		t.Fatal("bad key accepted")
	}
}

func TestToggleNestedAndRoot(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-n", map[string]string{"W/A/manifest.json": manifestJSON("X.A"), "W/B/manifest.json": manifestJSON("X.B")})
	e.item(t, "local-r", map[string]string{"manifest.json": manifestJSON("X.R"), "data.txt": "d"})
	p, _ := e.Create("stardew", "P")
	for _, k := range []string{"local-n", "local-r"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{}); err != nil {
			t.Fatal(err)
		}
	}
	mods := e.mods(p.ID)

	got, err := e.SetModEnabled("stardew", p.ID, "", "smapi:x.a", false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Entries[0].Disabled, []mod.ID{"smapi:X.A"}) {
		t.Fatalf("disabled = %v", got.Entries[0].Disabled)
	}
	if !slices.Equal(names(t, filepath.Join(mods, "local-n", "W")), []string{".A", "B"}) {
		t.Fatalf("W = %v", names(t, filepath.Join(mods, "local-n", "W")))
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.A", false); err != nil {
		t.Fatalf("not idempotent: %v", err)
	}

	got, err = e.SetModEnabled("stardew", p.ID, "", "smapi:X.R", false)
	if err != nil || !slices.Equal(got.Entries[1].Disabled, []mod.ID{"smapi:X.R"}) {
		t.Fatalf("root disable = %+v, %v", got, err)
	}
	if !slices.Equal(names(t, mods), []string{".local-r", "local-n"}) {
		t.Fatalf("mods = %v", names(t, mods))
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.R", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "local-r"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(t, mods), []string{"local-n"}) {
		t.Fatalf("after remove = %v", names(t, mods))
	}

	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.A", true); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.read("stardew", p.ID); len(got.Entries[0].Disabled) != 0 || !slices.Equal(names(t, filepath.Join(mods, "local-n", "W")), []string{"A", "B"}) {
		t.Fatalf("re-enable = %+v", got.Entries[0])
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.Nope", true); err == nil {
		t.Fatal("unknown mod accepted")
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "local-r"); err == nil {
		t.Fatal("removing a missing entry accepted")
	}
}

func TestSetModsEnabledAndRemoveEntriesOneWrite(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p, _ := e.Create("stardew", "P")
	for _, k := range []string{"local-a", "local-b"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.SetModsEnabled("stardew", p.ID, []EnableRef{
		{Key: "local-a", ID: "smapi:Me.A"},
		{Key: "local-b", ID: "smapi:Me.B"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Entries[0].Disabled, []mod.ID{"smapi:Me.A"}) || !slices.Equal(got.Entries[1].Disabled, []mod.ID{"smapi:Me.B"}) {
		t.Fatalf("disabled = %+v", got.Entries)
	}
	if !slices.Equal(names(t, e.mods(p.ID)), []string{".local-a", ".local-b"}) {
		t.Fatalf("mods = %v", names(t, e.mods(p.ID)))
	}
	got, err = e.RemoveEntries("stardew", p.ID, []string{"local-a", "local-b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 0 {
		t.Fatalf("entries = %+v", got.Entries)
	}
	if names := names(t, e.mods(p.ID)); len(names) != 0 {
		t.Fatalf("mods left = %v", names)
	}
}

func TestRootToggleRoundTrip(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-r", map[string]string{"manifest.json": manifestJSON("X.R")})
	p, _ := e.Create("stardew", "P")
	if _, err := e.AddEntry("stardew", p.ID, "local-r", Source{}); err != nil {
		t.Fatal(err)
	}
	for _, on := range []bool{false, true, false, true} {
		if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.R", on); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(names(t, e.mods(p.ID)), []string{"local-r"}) {
		t.Fatalf("mods = %v", names(t, e.mods(p.ID)))
	}
}

func TestDuplicateIsIndependent(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("X.A")})
	a, _ := e.Create("stardew", "A")
	b, _ := e.Create("stardew", "B")
	if _, err := e.AddEntry("stardew", a.ID, "local-a", Source{}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.mods(a.ID), "local-a/save.json", "orig")
	dup, err := e.Duplicate("stardew", a.ID)
	if err != nil || dup.Name != "A copy" || dup.ID == a.ID || len(dup.Entries) != 1 {
		t.Fatalf("dup = %+v, %v", dup, err)
	}
	list, _ := e.List("stardew")
	if len(list) != 3 || list[0].ID != a.ID || list[1].ID != dup.ID || list[2].ID != b.ID {
		t.Fatalf("order = %+v", list)
	}
	if b, _ := os.ReadFile(filepath.Join(e.mods(dup.ID), "local-a", "save.json")); string(b) != "orig" {
		t.Fatalf("copy lost mod data: %q", b)
	}
	writeFile(t, e.mods(dup.ID), "local-a/save.json", "changed")
	if b, _ := os.ReadFile(filepath.Join(e.mods(a.ID), "local-a", "save.json")); string(b) != "orig" {
		t.Fatalf("original changed: %q", b)
	}
	itemDir, _ := e.items.Path("stardew", "local-a")
	if b, _ := fsx.ReadFile(filepath.Join(itemDir, "manifest.json")); !strings.Contains(string(b), "X.A") {
		t.Fatal("store item changed")
	}
	long, _ := e.Create("stardew", strings.Repeat("é", 60))
	if d, err := e.Duplicate("stardew", long.ID); err != nil || len([]rune(d.Name)) > maxName {
		t.Fatalf("long dup = %q, %v", d.Name, err)
	}
}

func TestDuplicateUsesUniqueProfileName(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	source := mustCreate(t, s, "A")
	if _, err := s.Create("stardew", "A copy"); err != nil {
		t.Fatal(err)
	}

	dup, err := s.Duplicate("stardew", source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Name != "A copy (2)" {
		t.Fatalf("duplicate name = %q", dup.Name)
	}
}

func TestTrashRestorePurge(t *testing.T) {
	e := newEnv(t)
	a, _ := e.Create("stardew", "A")
	b, _ := e.Create("stardew", "B")
	if err := e.Delete("stardew", a.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := e.List("stardew"); len(list) != 1 || list[0].ID != b.ID {
		t.Fatalf("list = %+v", list)
	}
	tr, err := e.ListTrash("stardew")
	if err != nil || len(tr) != 1 || tr[0].ID != a.ID || tr[0].Name != "A" || tr[0].DaysLeft != 30 {
		t.Fatalf("trash = %+v, %v", tr, err)
	}
	if err := e.Delete("stardew", "../x"); err == nil {
		t.Fatal("bad id accepted")
	}
	if _, err := e.Restore("stardew", a.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := e.List("stardew"); len(list) != 2 {
		t.Fatalf("restored list = %+v", list)
	}
	if err := e.Delete("stardew", a.ID); err != nil {
		t.Fatal(err)
	}
	// Recreate the id's folder: restore must refuse.
	if err := os.MkdirAll(filepath.Join(e.root, "stardew", a.ID), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Restore("stardew", a.ID); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("restore over existing = %v", err)
	}
	if err := os.Remove(filepath.Join(e.root, "stardew", a.ID)); err != nil {
		t.Fatal(err)
	}

	if err := e.PurgeTrash(time.Now().Add(29 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if tr, _ := e.ListTrash("stardew"); len(tr) != 1 {
		t.Fatal("purged too early")
	}
	if err := e.PurgeTrash(time.Now().Add(31 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if tr, _ := e.ListTrash("stardew"); len(tr) != 0 {
		t.Fatalf("trash after purge = %+v", tr)
	}
}

func TestPurgeTrashUsesSettingsRetention(t *testing.T) {
	e := newEnv(t)
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Update(func(s *settings.Settings) { s.TrashRetentionDays = 1 }); err != nil {
		t.Fatal(err)
	}
	e.settings = st
	p := mustCreate(t, e, "P")
	if err := e.Delete("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.PurgeTrash(time.Now().Add(25 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if tr, _ := e.ListTrash("stardew"); len(tr) != 0 {
		t.Fatalf("1-day retention should have purged: %+v", tr)
	}
}

func TestPurgeTrash(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	if err := e.Delete("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(e.root, "outside")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := e.Purge("stardew", "../outside"); err == nil {
		t.Fatal("bad id accepted")
	}
	if err := e.Purge("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(e.root, "trash", "stardew", p.ID)) {
		t.Fatal("purged profile remains")
	}
	if !exists(outside) {
		t.Fatal("purge escaped trash")
	}
	q := mustCreate(t, e, "Q")
	if err := e.Delete("stardew", q.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.PurgeTrash("stardew"); err != nil {
		t.Fatal(err)
	}
	if trash, err := e.ListTrash("stardew"); err != nil || len(trash) != 0 {
		t.Fatalf("trash after empty = %+v, %v", trash, err)
	}
}

func TestRebuildAfterModsDeleted(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-n", map[string]string{"W/A/manifest.json": manifestJSON("X.A"), "W/B/manifest.json": manifestJSON("X.B")})
	e.item(t, "local-r", map[string]string{"manifest.json": manifestJSON("X.R")})
	p, _ := e.Create("stardew", "P")
	for _, k := range []string{"local-n", "local-r"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"X.A", "X.R"} {
		if _, err := e.SetModEnabled("stardew", p.ID, "", mod.SMAPI(id), false); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(e.mods(p.ID)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.mods(p.ID), ".tmp_leftover/x", "x")
	mods, err := e.Mods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(t, e.mods(p.ID)), []string{".local-r", "local-n"}) {
		t.Fatalf("mods = %v", names(t, e.mods(p.ID)))
	}
	if !slices.Equal(names(t, filepath.Join(e.mods(p.ID), "local-n", "W")), []string{".A", "B"}) {
		t.Fatal("disabled nested mod not dotted after rebuild")
	}
	if len(mods) != 3 || mods[0].Enabled || !mods[1].Enabled || !slices.Equal(mods[0].Siblings, []mod.ID{"smapi:X.B"}) || len(mods[2].Siblings) != 0 {
		t.Fatalf("mods = %+v", mods)
	}

	// One entry folder missing while the other stays untouched.
	writeFile(t, e.mods(p.ID), "local-n/W/B/keep.json", "kept")
	if err := os.RemoveAll(filepath.Join(e.mods(p.ID), ".local-r")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(t, e.mods(p.ID)), []string{".local-r", "local-n"}) {
		t.Fatalf("mods = %v", names(t, e.mods(p.ID)))
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), "local-n", "W", "B", "keep.json")); err != nil {
		t.Fatal("intact entry was recopied")
	}
}

func TestModsParkUnknownFoldersBeforeRebuild(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	writeFile(t, e.mods(p.ID), "dropped/keep.txt", "keep")

	if _, err := e.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UserMods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	held := filepath.Join(e.root, "stardew", p.ID, modsHoldDir, "dropped", "keep.txt")
	if got := read(t, held); got != "keep" {
		t.Fatalf("parked folder = %q", got)
	}
	if exists(filepath.Join(e.mods(p.ID), "dropped")) {
		t.Fatal("unknown folder remained in mods")
	}
}

func TestHiddenReorderAndStoreKeys(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("X.A")})
	a, _ := e.Create("stardew", "A")
	b, _ := e.Create("stardew", "B")
	c, _ := e.Create("stardew", "C")
	if p, err := e.SetHidden("stardew", b.ID, true); err != nil || !p.Hidden {
		t.Fatalf("hidden = %+v, %v", p, err)
	}
	if err := e.Reorder("stardew", []string{c.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	list, _ := e.List("stardew")
	if list[0].ID != c.ID || list[1].ID != a.ID || list[2].ID != b.ID || list[2].Order != 2 {
		t.Fatalf("order = %+v", list)
	}
	if err := e.Reorder("stardew", []string{a.ID, a.ID}); err == nil {
		t.Fatal("duplicate id accepted")
	}
	if err := e.Reorder("stardew", []string{"0000000000000000"}); err == nil {
		t.Fatal("unknown id accepted")
	}

	if _, err := e.AddEntry("stardew", a.ID, "local-a", Source{}); err != nil {
		t.Fatal(err)
	}
	if err := e.Delete("stardew", a.ID); err != nil {
		t.Fatal(err)
	}
	keys, err := e.StoreKeys(true)
	if err != nil || !slices.Equal(keys["stardew"], []string{"local-a"}) {
		t.Fatalf("keys = %v, %v", keys, err)
	}
}

func bundle() map[string]string {
	return map[string]string{
		"ConsoleCommands/manifest.json": manifestJSON("SMAPI.Console"),
		"SaveBackup/manifest.json":      manifestJSON("SMAPI.Backup"),
	}
}

func TestApplyBundledReplacesAndKeepsDisabled(t *testing.T) {
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	e.item(t, "smapi-2.0.0", bundle())
	e.item(t, "local-x", map[string]string{"manifest.json": manifestJSON("Other")})
	a := mustCreate(t, e, "a")
	b := mustCreate(t, e, "b")
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-1.0.0")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", a.ID, "local-x", Source{Kind: KindLocal, Name: "x.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", a.ID, "", "smapi:SMAPI.Backup", false); err != nil {
		t.Fatal(err)
	}
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-2.0.0")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ID, b.ID} {
		p, err := e.read("stardew", id)
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		for _, en := range p.Entries {
			keys = append(keys, en.Key)
			if en.Key == "smapi-2.0.0" && en.Source.Kind != SourceSMAPI {
				t.Fatalf("source = %+v", en.Source)
			}
		}
		want := []string{"smapi-2.0.0"}
		if id == a.ID {
			want = []string{"local-x", "smapi-2.0.0"}
		}
		if !slices.Equal(keys, want) {
			t.Fatalf("%s entries = %v, want %v", id, keys, want)
		}
	}
	got := names(t, filepath.Join(e.mods(a.ID), "smapi-2.0.0"))
	if !slices.Equal(got, []string{".SaveBackup", "ConsoleCommands"}) {
		t.Fatalf("a's bundled folders = %v (disabled state lost)", got)
	}
	if got := names(t, filepath.Join(e.mods(b.ID), "smapi-2.0.0")); !slices.Equal(got, []string{"ConsoleCommands", "SaveBackup"}) {
		t.Fatalf("b's bundled folders = %v", got)
	}
	if got := names(t, e.mods(a.ID)); !slices.Equal(got, []string{"local-x", "smapi-2.0.0"}) {
		t.Fatalf("old entry folder left behind: %v", got)
	}
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-2.0.0")); err != nil {
		t.Fatalf("reapplying the same key: %v", err)
	}
}

func TestCreateGetsBundledEntry(t *testing.T) {
	e := newEnv(t)
	key := ""
	e.Bundled = func(string) []Bundle {
		if key == "" {
			return nil
		}
		return []Bundle{smapiBundle(key)}
	}
	p, err := e.Create("stardew", "before")
	if err != nil || len(p.Entries) != 0 {
		t.Fatalf("before install: %+v, %v", p, err)
	}
	e.item(t, "smapi-1.0.0", bundle())
	key = "smapi-1.0.0"
	p, err = e.Create("stardew", "after")
	if err != nil || len(p.Entries) != 1 || p.Entries[0].Source.Kind != SourceSMAPI {
		t.Fatalf("after install: %+v, %v", p, err)
	}
	key = "smapi-0.0.1"
	if p, err = e.Create("stardew", "collected"); err != nil || len(p.Entries) != 0 {
		t.Fatalf("missing store item must not block creation: %+v, %v", p, err)
	}
}

func TestApplyBundledSkipsRunningProfile(t *testing.T) {
	e := newEnv(t)
	e.item(t, "smapi-1.0.0", bundle())
	e.item(t, "smapi-2.0.0", bundle())
	busy := mustCreate(t, e, "busy")
	idle := mustCreate(t, e, "idle")
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-1.0.0")); err != nil {
		t.Fatal(err)
	}
	e.Running = func(_, id string) bool { return id == busy.ID }
	if !e.AnyRunning("stardew") {
		t.Fatal("AnyRunning = false with a running profile")
	}
	var re *RunningError
	if err := e.ApplyBundled("stardew", smapiBundle("smapi-2.0.0")); !errors.As(err, &re) {
		t.Fatalf("err = %v, want a RunningError", err)
	}
	for id, want := range map[string]string{busy.ID: "smapi-1.0.0", idle.ID: "smapi-2.0.0"} {
		p, err := e.read("stardew", id)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Entries) != 1 || p.Entries[0].Key != want {
			t.Errorf("profile %s entries = %+v, want only %s", id, p.Entries, want)
		}
	}
	if err := e.ApplyBundledForStart("stardew", smapiBundle("smapi-2.0.0")); err != nil {
		t.Fatalf("during start: %v", err)
	}
	for _, id := range []string{busy.ID, idle.ID} {
		p, err := e.read("stardew", id)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Entries) != 1 || p.Entries[0].Key != "smapi-2.0.0" {
			t.Errorf("profile %s entries = %+v, want smapi-2.0.0", id, p.Entries)
		}
	}
}

func TestRunningProfileIsLocked(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "Locked")
	e.item(t, "a-1.0", map[string]string{"A/manifest.json": manifestJSON("me.a")})
	if _, err := e.AddEntry("stardew", p.ID, "a-1.0", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	running := true
	e.Running = func(game, id string) bool { return running && id == p.ID }

	var re *RunningError
	checks := map[string]error{}
	_, checks["AddEntry"] = e.AddEntry("stardew", p.ID, "a-1.0", Source{})
	_, checks["RemoveEntry"] = e.RemoveEntry("stardew", p.ID, "a-1.0")
	_, checks["RemoveEntries"] = e.RemoveEntries("stardew", p.ID, []string{"a-1.0"})
	_, checks["SetModEnabled"] = e.SetModEnabled("stardew", p.ID, "", "smapi:me.a", false)
	_, checks["SetModsEnabled"] = e.SetModsEnabled("stardew", p.ID, []EnableRef{{Key: "a-1.0", ID: "smapi:me.a"}}, false)
	checks["Delete"] = e.Delete("stardew", p.ID)
	_, checks["InstallArchive"] = e.InstallArchive("stardew", p.ID, "/nonexistent.zip")
	for name, err := range checks {
		if !errors.As(err, &re) || !strings.Contains(err.Error(), "Stardew Valley is running this profile") {
			t.Errorf("%s: err = %v, want a RunningError", name, err)
		}
	}
	if _, err := e.Rename("stardew", p.ID, "Renamed"); err != nil {
		t.Fatalf("rename does not touch mods/: %v", err)
	}

	running = false
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:me.a", false); err != nil {
		t.Fatal(err)
	}
	dir, err := e.ModsDir("stardew", p.ID)
	if err != nil || dir != e.mods(p.ID) {
		t.Fatalf("mods dir = %q, %v", dir, err)
	}
}

func TestUnreadableTrashedProfileBlocksOnlyItself(t *testing.T) {
	e := newEnv(t)
	good, _ := e.Create("stardew", "Good")
	bad, _ := e.Create("stardew", "Bad")
	for _, id := range []string{good.ID, bad.ID} {
		if err := e.Delete("stardew", id); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(e.trash, "stardew", bad.ID), fileName, "{not json")
	if tr, err := e.ListTrash("stardew"); err != nil || len(tr) != 1 || tr[0].ID != good.ID {
		t.Fatalf("trash = %+v, %v", tr, err)
	}
	if err := e.PurgeTrash(time.Now().Add(31 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.trash, "stardew", good.ID)); err == nil {
		t.Fatal("readable expired profile was not purged")
	}
	if _, err := e.StoreKeys(true); err == nil {
		t.Fatal("StoreKeys ignored an unreadable trashed profile")
	}
}

// A running check made before the lock lets a launch start between the check and the change.
func TestRunningIsCheckedUnderTheLock(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "A")
	s.Running = func(string, string) bool {
		if s.mu.TryLock() {
			s.mu.Unlock()
			t.Error("running checked without the store lock")
		}
		return true
	}
	ops := map[string]func() error{
		"AddEntry":      func() error { _, err := s.AddEntry("stardew", p.ID, "k", Source{}); return err },
		"UpdateEntry":   func() error { _, err := s.UpdateEntry("stardew", p.ID, "a", "b"); return err },
		"RemoveEntry":   func() error { _, err := s.RemoveEntry("stardew", p.ID, "k"); return err },
		"RemoveEntries": func() error { _, err := s.RemoveEntries("stardew", p.ID, []string{"k"}); return err },
		"SetModEnabled": func() error { _, err := s.SetModEnabled("stardew", p.ID, "", "smapi:x", false); return err },
		"SetModsEnabled": func() error {
			_, err := s.SetModsEnabled("stardew", p.ID, []EnableRef{{ID: "smapi:x"}}, false)
			return err
		},
		"Delete": func() error { return s.Delete("stardew", p.ID) },
	}
	for name, op := range ops {
		var re *RunningError
		if err := op(); !errors.As(err, &re) {
			t.Errorf("%s: err = %v, want RunningError", name, err)
		}
	}
}
