package bundles

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/profile"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/testenv"
)

func testService(t *testing.T) (*Service, *profile.Store, *modstore.Store) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("GOTMPDIR", tmp)
	dataHome := filepath.Join(tmp, "xdg")
	if err := os.MkdirAll(dataHome, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", dataHome)
	items, profiles := testenv.Stores(t)
	dataDir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	return NewService(profiles, dataDir), profiles, items
}

func addItem(t *testing.T, items *modstore.Store, key, uniqueID, name string) {
	t.Helper()
	src := t.TempDir()
	body := `{"Name":"` + name + `","UniqueID":"` + uniqueID + `","Version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir("stardew", key, src); err != nil {
		t.Fatal(err)
	}
}

func TestBundleStoreSnapshotsAndPersistsMods(t *testing.T) {
	svc, profiles, items := testService(t)
	addItem(t, items, "local-a", "A.One", "A One")
	addItem(t, items, "local-b", "B.Two", "B Two")
	source, err := profiles.Create("stardew", "Source")
	if err != nil {
		t.Fatal(err)
	}
	source, err = profiles.AddEntry("stardew", source.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", source.ID, "local-b", profile.Source{Kind: profile.KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}

	b, err := svc.Create("stardew", " SVE core ", source.ID, []string{"B.Two", "A.One"})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == "" || b.Name != "SVE core" || len(b.Mods) != 2 {
		t.Fatalf("created bundle = %+v", b)
	}
	if b.Mods[0].EntryKey != "local-b" || b.Mods[0].Source.Name != "b.zip" {
		t.Fatalf("first snapshot = %+v", b.Mods[0])
	}
	list, err := svc.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !slices.Equal([]string{list[0].Mods[0].UniqueID, list[0].Mods[1].UniqueID}, []string{"B.Two", "A.One"}) {
		t.Fatalf("listed bundles = %+v", list)
	}
	b, err = svc.Rename("stardew", b.ID, "SVE essentials")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "SVE essentials" {
		t.Fatalf("renamed bundle = %+v", b)
	}
	if err := svc.Delete("stardew", b.ID); err != nil {
		t.Fatal(err)
	}
	list, err = svc.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("bundles after delete = %+v", list)
	}
}

func TestReferencedKeysKeepProfileExtrasAndBundlesDuringCollect(t *testing.T) {
	svc, profiles, items := testService(t)
	addItem(t, items, "profile-main", "Profile.Main", "Profile Main")
	addItem(t, items, "profile-extra", "Profile.Extra", "Profile Extra")
	addItem(t, items, "bundle-only", "Bundle.Only", "Bundle Only")
	addItem(t, items, "unused", "Unused.Mod", "Unused")

	profileSource, err := profiles.Create("stardew", "Profile")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", profileSource.ID, "profile-main", profile.Source{}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddExtra("stardew", profileSource.ID, "profile-main", "profile-extra", profile.Source{}); err != nil {
		t.Fatal(err)
	}

	bundleSource, err := profiles.Create("stardew", "Bundle source")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", bundleSource.ID, "bundle-only", profile.Source{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create("stardew", "Saved bundle", bundleSource.ID, []string{"Bundle.Only"}); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Delete("stardew", bundleSource.ID); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Purge("stardew", bundleSource.ID); err != nil {
		t.Fatal(err)
	}

	keys, err := profiles.StoreKeys()
	if err != nil {
		t.Fatal(err)
	}
	bundleKeys, err := svc.ReferencedStoreKeys()
	if err != nil {
		t.Fatal(err)
	}
	for game, referenced := range bundleKeys {
		keys[game] = append(keys[game], referenced...)
	}
	if err := items.Collect(keys, time.Now().Add(31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"profile-main", "profile-extra", "bundle-only"} {
		if _, err := items.Path("stardew", key); err != nil {
			t.Fatalf("referenced item %q was collected: %v", key, err)
		}
	}
	if _, err := items.Path("stardew", "unused"); !errors.Is(err, modstore.ErrNotFound) {
		t.Fatalf("unreferenced item remains: %v", err)
	}
}

func TestApplyAddsStoreEntriesAndReportsMissingMods(t *testing.T) {
	svc, profiles, items := testService(t)
	addItem(t, items, "local-a", "A.One", "A One")
	addItem(t, items, "local-b", "B.Two", "B Two")
	source, err := profiles.Create("stardew", "Source")
	if err != nil {
		t.Fatal(err)
	}
	source, err = profiles.AddEntry("stardew", source.ID, "local-a", profile.Source{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", source.ID, "local-b", profile.Source{}); err != nil {
		t.Fatal(err)
	}
	target, err := profiles.Create("stardew", "Target")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create("stardew", "Bundle", source.ID, []string{"A.One", "B.Two"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(mustPath(t, items, "stardew", "local-b")); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Apply("stardew", b.ID, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || !slices.Equal(result.Missing, []string{"B Two"}) {
		t.Fatalf("apply result = %+v", result)
	}
	if len(result.Profile.Entries) != 1 || result.Profile.Entries[0].Key != "local-a" {
		t.Fatalf("applied profile = %+v", result.Profile)
	}

	profiles.Running = func(_, id string) bool { return id == target.ID }
	if _, err := svc.Apply("stardew", b.ID, target.ID); !errors.As(err, new(*profile.RunningError)) {
		t.Fatalf("running apply = %v", err)
	}
}

func mustPath(t *testing.T, items *modstore.Store, game, key string) string {
	t.Helper()
	path, err := items.Path(game, key)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
