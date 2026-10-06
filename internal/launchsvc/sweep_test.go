package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func sweepEnv(t *testing.T) (*Service, *profile.Store) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	svc := NewService(t.TempDir(), set, profiles)
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.15", "4.1.10", nil }
	svc.SweepCompat = func(context.Context) (meta.CompatIndex, error) {
		return meta.CompatIndex{ByID: map[string]meta.CompatEntry{}}, nil
	}
	svc.SweepHasUpdate = func(mod.ID, int) bool { return false }
	return svc, profiles
}

func addSweepMod(t *testing.T, profiles *profile.Store, profileID, uniqueID, name string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := `{"Name":"` + name + `","Author":"A","Version":"1.0.0","UniqueID":"` + uniqueID + `"}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := profiles.ImportExternalMods("stardew", profileID, []profile.ExternalMod{
		{SourcePath: dir, ID: mod.SMAPI(uniqueID), Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSweepNoChangeSkips(t *testing.T) {
	svc, _ := sweepEnv(t)
	if err := svc.settings.RecordLastSweep("stardew", "1.6.15", "smapi", "4.1.10"); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.Sweep(context.Background(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Triggered {
		t.Fatalf("unchanged versions triggered: %#v", rep)
	}
	if len(rep.Profiles) != 0 {
		t.Fatalf("no-change sweep listed profiles: %#v", rep.Profiles)
	}
}

func TestSweepBrokenWithAndWithoutFix(t *testing.T) {
	svc, profiles := sweepEnv(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	addSweepMod(t, profiles, p.ID, "A.Fixable", "Fixable")
	addSweepMod(t, profiles, p.ID, "A.Stuck", "Stuck")
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.16", "4.2.0", nil }
	svc.SweepCompat = func(context.Context) (meta.CompatIndex, error) {
		return meta.CompatIndex{ByID: map[string]meta.CompatEntry{
			"a.fixable": {Status: meta.StatusBroken, Summary: "Use latest version"},
			"a.stuck":   {Status: meta.StatusBroken, Summary: "Broken in 1.6.16"},
		}}, nil
	}
	svc.SweepHasUpdate = func(uniqueID mod.ID, _ int) bool {
		return uniqueID == "smapi:A.Fixable"
	}
	rep, err := svc.Sweep(context.Background(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Triggered {
		t.Fatal("version change did not trigger")
	}
	if len(rep.Profiles) != 1 {
		t.Fatalf("profiles = %#v", rep.Profiles)
	}
	byID := map[string]SweepMod{}
	for _, m := range rep.Profiles[0].Broken {
		byID[m.ID.Local()] = m
	}
	if byID["A.Fixable"].Fix != fixUpdate {
		t.Fatalf("fixable = %#v", byID["A.Fixable"])
	}
	if byID["A.Stuck"].Fix != fixNone {
		t.Fatalf("stuck = %#v", byID["A.Stuck"])
	}
}
