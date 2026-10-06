package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// pkgWith seeds the scan cache for a package that ships one plugin with the given relations.
func pkgWith(key, guid, version string, enabled bool, relations ...dotnet.Relation) profile.PackageRef {
	dir := "/pkg/" + key
	for i := range relations {
		relations[i].Plugin = guid
	}
	pluginMu.Lock()
	pluginCache[pluginKey{dir}] = dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: guid, Name: key, Version: version}}, Relations: relations}
	pluginMu.Unlock()
	return profile.PackageRef{Key: key, Enabled: enabled, ID: mod.NewID(mod.FormatBepInEx, key), Name: key, Version: version, Dir: dir}
}

func hard(guid, minimum string) dotnet.Relation {
	return dotnet.Relation{GUID: guid, MinVersion: minimum, Kind: dotnet.HardDependency}
}

func TestPluginDepsHardDependencyIsMetAbsentOrTooOld(t *testing.T) {
	lib := pkgWith("lib", "com.x.Lib", "2.0.0", true)
	met := pkgWith("met", "com.x.Met", "1.0.0", true, hard("COM.X.LIB", "1.5.0"))
	old := pkgWith("old", "com.x.Old", "1.0.0", true, hard("com.x.lib", "2.1.0"))
	gone := pkgWith("gone", "com.x.Gone", "1.0.0", true, hard("com.x.nowhere", ""))

	got, _ := pluginDeps([]profile.PackageRef{lib, met, old, gone}, nil)

	if len(got) != 2 {
		t.Fatalf("missing = %+v, want old and gone only", got)
	}
	if o := got[0]; o.DependentName != "old" || o.Reason != "outdated" || o.InstalledVersion != "2.0.0" || o.MinimumVersion != "2.1.0" || o.ID != lib.ID {
		t.Fatalf("outdated row = %+v", o)
	}
	if g := got[1]; g.DependentName != "gone" || g.Reason != "absent" || g.ID != "bepinex:com.x.nowhere" || g.Optional {
		t.Fatalf("absent row = %+v", g)
	}
}

func TestPluginDepsNamesALibraryThatIsDisabled(t *testing.T) {
	lib := pkgWith("lib", "com.x.Lib", "2.0.0", false)
	user := pkgWith("user", "com.x.User", "1.0.0", true, hard("com.x.lib", "1.0.0"))

	got, _ := pluginDeps([]profile.PackageRef{lib, user}, nil)

	if len(got) != 1 || got[0].Reason != "disabled" || got[0].ID != lib.ID {
		t.Fatalf("missing = %+v, want the disabled library named by its package id", got)
	}
}

func TestPluginDepsAnAbsentSoftDependencyIsOptional(t *testing.T) {
	user := pkgWith("user", "com.x.User", "1.0.0", true, dotnet.Relation{GUID: "com.x.extra", Kind: dotnet.SoftDependency})
	off := pkgWith("off", "com.x.Off", "1.0.0", true, dotnet.Relation{GUID: "com.x.sleeping", Kind: dotnet.SoftDependency})
	sleeping := pkgWith("sleeping", "com.x.Sleeping", "1.0.0", false)

	got, _ := pluginDeps([]profile.PackageRef{user, off, sleeping}, nil)

	if len(got) != 1 || got[0].DependentName != "user" || !got[0].Optional || got[0].Reason != "absent" {
		t.Fatalf("missing = %+v, want one optional row; a disabled soft dependency is the player's choice", got)
	}
	if (Result{Missing: got}).Count() != 0 {
		t.Fatal("an optional row is not counted")
	}
}

func TestPluginDepsReportsAnIncompatibilityBetweenEnabledPackages(t *testing.T) {
	a := pkgWith("a", "com.x.A", "1.0.0", true, dotnet.Relation{GUID: "com.x.b", Kind: dotnet.Incompatible})
	b := pkgWith("b", "com.x.B", "1.0.0", true)
	c := pkgWith("c", "com.x.C", "1.0.0", true, dotnet.Relation{GUID: "com.x.d", Kind: dotnet.Incompatible})
	d := pkgWith("d", "com.x.D", "1.0.0", false)

	_, got := pluginDeps([]profile.PackageRef{a, b, c, d}, nil)

	if len(got) != 1 || got[0].Key != "a" || got[0].Kind != "incompatible-plugin" || got[0].Dependency != "b" {
		t.Fatalf("failures = %+v, want a against b; c's partner is disabled", got)
	}
}

func TestPluginDepsSkipsTheLoaderAndAModShippedUnderAnotherGUID(t *testing.T) {
	user := pkgWith("user", "com.x.User", "1.0.0", true,
		hard("BepInEx", ""), hard("BepInEx.Harmony", ""), hard("com.rune580.lethalcompanyinpututils", ""), hard("com.x.user", ""))
	named := inst("inpututils", "LethalCompany_InputUtils", "1.0.0", true)

	got, _ := pluginDeps([]profile.PackageRef{user}, []framework.Mod{named})

	if len(got) != 0 {
		t.Fatalf("missing = %+v, want none", got)
	}
}

func TestPluginDepsReadsTheAttributesOfARealAssembly(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(dir, "Plugin.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := profile.PackageRef{Key: "fixture", Enabled: true, ID: "bepinex:fixture", Name: "Fixture", Dir: dir}
	clash := pkgWith("clash", "com.fixture.clash", "1.0.0", true)

	missing, failures := pluginDeps([]profile.PackageRef{pkg, clash}, nil)

	if len(missing) != 3 || len(failures) != 1 || failures[0].Dependency != "clash" {
		t.Fatalf("missing = %+v, failures = %+v", missing, failures)
	}
	if missing[1].MinimumVersion != "2.1.0" || !missing[2].Optional {
		t.Fatalf("missing = %+v", missing)
	}
}

func TestProblemsCountsAnInstalledPluginsMissingHardDependency(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"Fixture","version_number":"1.0.0","dependencies":[]}`, "Fixture.dll": string(dll),
	})
	if _, err := profiles.InstallSource("lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore"})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(t.TempDir(), set, profiles, nil)
	s.meta = fakeMeta{}

	got, err := s.Problems(t.Context(), "lethal-company", p.ID)

	if err != nil || len(got.Missing) != 3 || got.Count() != 2 {
		t.Fatalf("missing = %+v, count = %d, err = %v; want the hard and the minimum-version dependency counted and the soft one not", got.Missing, got.Count(), err)
	}
}
