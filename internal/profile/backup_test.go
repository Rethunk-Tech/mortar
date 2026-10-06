package profile

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
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

	got, missing, err := e.RestoreBackup("stardew", src, files, nil)
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
	_, missing, err = fresh.RestoreBackup("stardew", src, files, nil)
	if err != nil || len(missing) != 2 {
		t.Fatalf("a computer without the store items must download both: %v, %v", missing, err)
	}
	if _, _, err := e.RestoreBackup("stardew", src, map[string][]byte{"../escape.json": nil}, nil); err == nil {
		t.Fatal("a path outside the profile must be refused")
	}
}
