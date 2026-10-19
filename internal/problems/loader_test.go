package problems

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestLoaderFailuresAttributeLogFindingsToPackages(t *testing.T) {
	log, err := fsx.ReadFile(filepath.Join("..", "loader", "bepinex5", "testdata", "logoutput-missing-dependency.log"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "BepInEx/LogOutput.log", string(log))
	boombox := inst("boombox", "YoutubeBoombox", "1.0", true)
	player := "NullReferenceException: boom\n  at Boombox.Plugin.Update () [0x00000] in <abc>:0\n"
	known := map[string]framework.Mod{"youtube boombox 1.5.0": boombox}
	for name, owners := range map[string]map[string]framework.Mod{"known": known, "unknown": {}} {
		got := loaderFailures(bepinex5.Loader{}, loader.ProfileView{Dir: dir}, player, func() map[string]framework.Mod { return owners })
		if len(got) != 3 || got[0].Plugin != "Youtube Boombox 1.5.0" || got[0].Kind != bepinex5.KindMissingDependency {
			t.Fatalf("%s: failures = %+v", name, got)
		}
		wantKey := ""
		if name == "known" {
			wantKey = "boombox"
		}
		if got[0].Key != wantKey || got[0].Name != boombox.Name && wantKey != "" {
			t.Fatalf("%s: owner = %q %q", name, got[0].Key, got[0].Name)
		}
	}
}

func TestGameVersionFailureNeedsAnOlderGameThanTheLoaderAccepts(t *testing.T) {
	l, _ := game.PrimaryLoader("stardew")
	for _, c := range []struct {
		version string
		bad     bool
	}{{"1.5.6", true}, {"1.6.14", false}, {"", false}} {
		_, bad := gameVersionFailure("stardew", l, Environment{GameVersion: c.version, VersionScheme: deps.SemverSMAPI})
		if bad != c.bad {
			t.Errorf("game %q flagged = %v", c.version, bad)
		}
	}
}

func TestPluginOwnersNameAPackageByItsPluginsNamespace(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	pkg := func(key string) framework.Mod {
		m := inst(key, key, "1.0.0", true)
		m.Folder = t.TempDir()
		if err := fsx.WriteFile(filepath.Join(m.Folder, "Plugin.dll"), dll, 0o600); err != nil {
			t.Fatal(err)
		}
		return m
	}
	one := pkg("one")
	if got := pluginOwners([]framework.Mod{one})["fixture"]; got.Key != "one" {
		t.Fatalf("the plugin class's root namespace names its package: %+v", got)
	}
	if got, ok := pluginOwners([]framework.Mod{one, pkg("two")})["fixture"]; ok {
		t.Fatalf("a namespace two packages share names neither: %+v", got)
	}
}

func TestStartFailureIsALoadFailureOnlyForARunThatNeverStarted(t *testing.T) {
	got := startFailure("run-1", launch.Summary{StartFailure: "the game did not start in time"})
	if len(got) != 1 || got[0].Kind != KindRunFailed || got[0].Plugin != "run-1" || got[0].Message != "the game did not start in time" {
		t.Fatalf("failures = %+v", got)
	}
	if got := startFailure("run-2", launch.Summary{Errors: 3}); len(got) != 0 {
		t.Fatalf("a run that started is not a start failure: %+v", got)
	}
}

func TestSkippedOlderCopyNoPackageOwnsGetsARemovablePath(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "BepInEx/LogOutput.log",
		"[14:35:39.1094737] [Warning:   BepInEx] Skipping [Fixture Plugin 1.2.3] because a newer version exists (Fixture Plugin 1.3.0)\n")
	testfs.WriteFile(t, dir, "BepInEx/plugins/old/Fixture.dll", string(dll))
	none := func() map[string]framework.Mod { return nil }
	got := loaderFailures(bepinex5.Loader{}, loader.ProfileView{Dir: dir}, "", none)
	if len(got) != 1 || got[0].Kind != bepinex5.KindDuplicateGUID || got[0].Path != "BepInEx/plugins/old/Fixture.dll" {
		t.Fatalf("failures = %+v", got)
	}
	owned := func() map[string]framework.Mod {
		return map[string]framework.Mod{"fixture plugin 1.2.3": inst("k", "Fixture", "1.2.3", true)}
	}
	if got := loaderFailures(bepinex5.Loader{}, loader.ProfileView{Dir: dir}, "", owned); len(got) != 1 || got[0].Path != "" {
		t.Fatalf("an owned copy offered a file to delete: %+v", got)
	}
}

func TestRemoveSkippedPluginDeletesOnlyAPluginDLLInTheProfile(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	dir, err := profiles.ProfileDir("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "BepInEx/plugins/old/Fixture.dll", "x")
	testfs.WriteFile(t, dir, "BepInEx/config/keep.cfg", "x")
	s := &Service{profiles: profiles, cache: map[string]cached{}}
	for _, bad := range []string{"BepInEx/config/keep.cfg", "BepInEx/plugins/../config/keep.cfg", "../x.dll", "BepInEx/plugins/old"} {
		if err := s.RemoveSkippedPlugin("lethal-company", p.ID, bad); err == nil {
			t.Fatalf("%q was removable", bad)
		}
	}
	if err := s.RemoveSkippedPlugin("lethal-company", p.ID, "BepInEx/plugins/old/Fixture.dll"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx/plugins/old/Fixture.dll")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the DLL is still there: %v", err)
	}
}

// A dependency lands before the mod that needs it, so during a queue install each implicit mod it brings reads as
// unneeded; the count must not jump up and fall back when the mod arrives.
func TestOrphanRowsOfNewDependenciesWaitForTheQueueToDrain(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore"})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	install := func(name string, implicit bool) {
		short := name[strings.Index(name, "-")+1:]
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
			"manifest.json": `{"name":"` + short + `","version_number":"1.0.0","dependencies":[]}`, "F.txt": name,
		})
		src := profile.Source{Kind: profile.KindThunderstore, Name: name, Version: "1.0.0"}.WithImplicit(implicit)
		if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, src); err != nil {
			t.Fatal(err)
		}
	}
	installing := false
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(t.TempDir(), set, profiles, nil)
	s.Installing = func(string, string) bool { return installing }
	count := func() int {
		r, err := s.Problems(t.Context(), "lethal-company", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(r.Cleanup)
	}
	install("Ns-Root", false)
	install("Ns-Old", true)
	before := count()
	installing = true
	install("Ns-Dep", true)
	if got := count(); got != before {
		t.Fatalf("a dependency landing mid-install changed the count from %d to %d", before, got)
	}
	installing = false
	if got := count(); got != before+1 {
		t.Fatalf("after the queue drained: %d, want %d", got, before+1)
	}
}
