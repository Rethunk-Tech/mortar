package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
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
func pkgWith(t *testing.T, key, guid, version string, enabled bool, relations ...dotnet.Relation) profile.PackageRef {
	dir := filepath.Join(t.TempDir(), key)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
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
	lib := pkgWith(t, "lib", "com.x.Lib", "2.0.0", true)
	met := pkgWith(t, "met", "com.x.Met", "1.0.0", true, hard("COM.X.LIB", "1.5.0"))
	old := pkgWith(t, "old", "com.x.Old", "1.0.0", true, hard("com.x.lib", "2.1.0"))
	gone := pkgWith(t, "gone", "com.x.Gone", "1.0.0", true, hard("com.x.nowhere", ""))

	got, _ := pluginDeps([]profile.PackageRef{lib, met, old, gone}, nil)

	if len(got) != 2 {
		t.Fatalf("missing = %+v, want old and gone only", got)
	}
	if o := got[0]; o.DependentName != "old" || o.Reason != "outdated" || o.InstalledVersion != "2.0.0" || o.MinimumVersion != "2.1.0" || o.ID != lib.ID {
		t.Fatalf("outdated row = %+v", o)
	}
	if g := got[1]; g.DependentName != "gone" || g.Reason != "absent" || g.ID != "bepinex:com.x.nowhere" {
		t.Fatalf("absent row = %+v", g)
	}
}

func TestPluginDepsNamesALibraryThatIsDisabled(t *testing.T) {
	lib := pkgWith(t, "lib", "com.x.Lib", "2.0.0", false)
	user := pkgWith(t, "user", "com.x.User", "1.0.0", true, hard("com.x.lib", "1.0.0"))

	got, _ := pluginDeps([]profile.PackageRef{lib, user}, nil)

	if len(got) != 1 || got[0].Reason != "disabled" || got[0].ID != lib.ID {
		t.Fatalf("missing = %+v, want the disabled library named by its package id", got)
	}
}

func TestPluginDepsIgnoresSoftDependencies(t *testing.T) {
	user := pkgWith(t, "user", "com.x.User", "1.0.0", true, dotnet.Relation{GUID: "com.x.extra", Kind: dotnet.SoftDependency})
	off := pkgWith(t, "off", "com.x.Off", "1.0.0", true, dotnet.Relation{GUID: "com.x.sleeping", Kind: dotnet.SoftDependency})
	sleeping := pkgWith(t, "sleeping", "com.x.Sleeping", "1.0.0", false)

	got, _ := pluginDeps([]profile.PackageRef{user, off, sleeping}, nil)

	if len(got) != 0 {
		t.Fatalf("missing = %+v, want none: a soft dependency is an optional integration hook", got)
	}
}

func TestPluginDepsReportsAnIncompatibilityBetweenEnabledPackages(t *testing.T) {
	a := pkgWith(t, "a", "com.x.A", "1.0.0", true, dotnet.Relation{GUID: "com.x.b", Kind: dotnet.Incompatible})
	b := pkgWith(t, "b", "com.x.B", "1.0.0", true)
	c := pkgWith(t, "c", "com.x.C", "1.0.0", true, dotnet.Relation{GUID: "com.x.d", Kind: dotnet.Incompatible})
	d := pkgWith(t, "d", "com.x.D", "1.0.0", false)

	_, got := pluginDeps([]profile.PackageRef{a, b, c, d}, nil)

	if len(got) != 1 || got[0].Key != "a" || got[0].Kind != "incompatible-plugin" || got[0].Dependency != "b" {
		t.Fatalf("failures = %+v, want a against b; c's partner is disabled", got)
	}
}

func TestPluginDepsSkipsTheLoaderAndAModShippedUnderAnotherGUID(t *testing.T) {
	user := pkgWith(t, "user", "com.x.User", "1.0.0", true,
		hard("BepInEx", ""), hard("BepInEx.Harmony", ""), hard("com.rune580.lethalcompanyinpututils", ""), hard("com.x.user", ""))
	named := inst("inpututils", "LethalCompany_InputUtils", "1.0.0", true)

	got, _ := pluginDeps([]profile.PackageRef{user}, []framework.Mod{named})

	if len(got) != 0 {
		t.Fatalf("missing = %+v, want none", got)
	}
}

func TestPluginDepsReadsTheAttributesOfARealAssembly(t *testing.T) {
	dll, err := safefs.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := safefs.WriteFile(filepath.Join(dir, "Plugin.dll"), dll, 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := profile.PackageRef{Key: "fixture", Enabled: true, ID: "bepinex:fixture", Name: "Fixture", Dir: dir}
	clash := pkgWith(t, "clash", "com.fixture.clash", "1.0.0", true)

	missing, failures := pluginDeps([]profile.PackageRef{pkg, clash}, nil)

	if len(missing) != 2 || len(failures) != 1 || failures[0].Dependency != "clash" {
		t.Fatalf("missing = %+v, failures = %+v", missing, failures)
	}
	if missing[1].MinimumVersion != "2.1.0" {
		t.Fatalf("missing = %+v", missing)
	}
}

func TestProblemsCountsAnInstalledPluginsMissingHardDependency(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	dll, err := safefs.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"Fixture","version_number":"1.0.0","dependencies":[]}`, "Fixture.dll": string(dll),
	})
	if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"}); err != nil {
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

	if err != nil || len(got.Missing) != 2 || got.Count() != 2 {
		t.Fatalf("missing = %+v, count = %d, err = %v; want the hard and the minimum-version dependency, both counted", got.Missing, got.Count(), err)
	}
}

// manifestProfile installs Thunderstore packages with manifests into a Lethal Company profile and returns what Problems
// says about it.
type manifestPkg struct {
	name, version, deps string
}

func manifestProblems(t *testing.T, disable []string, pkgs ...manifestPkg) Result {
	t.Helper()
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	for _, pk := range pkgs {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
			"manifest.json": `{"name":"` + pk.name + `","version_number":"` + pk.version + `","dependencies":[` + pk.deps + `]}`, pk.name + ".dll": "x",
		})
		res, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + pk.name, Version: pk.version})
		if err != nil {
			t.Fatal(err)
		}
		for _, off := range disable {
			if off == pk.name {
				e := res.Profile.Entries[len(res.Profile.Entries)-1]
				if _, err := profiles.SetModEnabled("lethal-company", p.ID, e.Key, e.Mods[0].ID, false); err != nil {
					t.Fatal(err)
				}
			}
		}
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
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestThunderstoreManifestDependenciesAreMissingDisabledOrOutdated(t *testing.T) {
	got := manifestProblems(t, []string{"Off"},
		manifestPkg{"Lib", "1.0.0", ""},
		manifestPkg{"Off", "1.0.0", ""},
		manifestPkg{"Needy", "1.0.0", `"BepInEx-BepInExPack-5.4.2100","Ns-Gone-1.0.0","Ns-Off-1.0.0","Ns-Lib-2.0.0"`},
	)

	reasons := map[string]string{}
	for _, m := range got.Missing {
		reasons[m.ID.Local()] = m.Reason
	}
	want := map[string]string{"Ns-Gone": "absent", "Ns-Off": "disabled", "Ns-Lib": "outdated"}
	if len(reasons) != 3 || reasons["Ns-Gone"] != want["Ns-Gone"] || reasons["Ns-Off"] != want["Ns-Off"] || reasons["Ns-Lib"] != want["Ns-Lib"] {
		t.Fatalf("missing = %+v, want %v and no row for the loader pack", got.Missing, want)
	}
}

func TestDropCoveredPrefersTheManifestsPackageNaming(t *testing.T) {
	dependent := mod.NewID(mod.FormatThunderstore, "Ns-User")
	manifest := []Missing{
		{DependentID: dependent, ID: "thunderstore:Evaisa-LethalLib", Reason: "absent"},
		{DependentID: dependent, ID: "thunderstore:Ns-Lib", Reason: "disabled"},
	}
	dll := []Missing{
		{DependentID: dependent, ID: "bepinex:evaisa.lethallib", Reason: "absent"},
		{DependentID: dependent, ID: "bepinex:com.sigurd.csync", Reason: "absent"},
		{DependentID: dependent, ID: "thunderstore:Ns-Lib", Reason: "disabled"},
		{DependentID: "thunderstore:Ns-Other", ID: "bepinex:evaisa.lethallib", Reason: "absent"},
	}

	got := dropCovered(dll, manifest)

	if len(got) != 2 || got[0].ID != "bepinex:com.sigurd.csync" || got[1].DependentID != "thunderstore:Ns-Other" {
		t.Fatalf("rows = %+v, want the CSync row and the other dependent's row", got)
	}
}

func TestProblemsReportsAMissingPackageOnceWhenManifestAndAssemblyBothNameIt(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	dll, err := safefs.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"Fixture","version_number":"1.0.0","dependencies":["Ns-Hard-1.0.0"]}`, "Fixture.dll": string(dll),
	})
	if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"}); err != nil {
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

	ids := map[mod.ID]bool{}
	for _, m := range got.Missing {
		ids[m.ID] = true
	}
	if err != nil || len(got.Missing) != 2 || !ids["thunderstore:Ns-Hard"] || !ids["bepinex:com.fixture.min"] {
		t.Fatalf("missing = %+v, err = %v; want Ns-Hard by package once, and the version-pinned GUID", got.Missing, err)
	}
}

func TestAPluginsAttributesAddItsDependenciesSoItsLibrariesAreNotOfferedForRemoval(t *testing.T) {
	lib := pkgWith(t, "lethallib", "evaisa.lethallib", "1.0.0", true)
	extra := pkgWith(t, "extra", "com.x.Extra", "1.0.0", true)
	clash := pkgWith(t, "clash", "com.x.Clash", "1.0.0", true)
	// A locally built mod with no manifest: its plugin's attributes are all that name its libraries.
	local := pkgWith(t, "local", "com.wallet.Local", "0.1.0", true,
		hard("evaisa.lethallib", ""),
		dotnet.Relation{GUID: "com.x.extra", Kind: dotnet.SoftDependency},
		dotnet.Relation{GUID: "com.x.clash", Kind: dotnet.Incompatible})
	pkgs := []profile.PackageRef{lib, extra, clash, local}
	mods := make([]framework.Mod, len(pkgs))
	for i, p := range pkgs {
		mods[i] = framework.Mod{Key: p.Key, Enabled: true, Implicit: p.Key == "lethallib", Manifest: smapimanifest.Manifest{Name: p.Name}.WithModID(p.ID)}
	}

	withPluginDependencies(mods, pkgs)

	deps := mods[3].Dependencies
	if len(deps) != 2 || !mod.Equal(deps[0].ModID(), lib.ID) || !deps[0].Required || !mod.Equal(deps[1].ModID(), extra.ID) || deps[1].Required {
		t.Fatalf("local mod's dependencies = %+v, want LethalLib required and Extra optional", deps)
	}
	if hints := cleanupHints(mods, nil, nil, nil, nil); len(hints) != 0 {
		t.Fatalf("cleanup = %+v, want LethalLib kept for the local mod", hints)
	}
}

func TestPluginDepsReadsPluginVersionsAsSystemVersion(t *testing.T) {
	lib := pkgWith(t, "lib", "com.x.Lib", "1.0", true)
	beta := pkgWith(t, "beta", "com.x.Beta", "1.0.0-beta", true)
	a := pkgWith(t, "a", "com.x.A", "1.0.0", true, hard("com.x.lib", "1.0.0"))
	b := pkgWith(t, "b", "com.x.B", "1.0.0", true, hard("com.x.beta", ""))

	got, _ := pluginDeps([]profile.PackageRef{lib, beta, a, b}, nil)

	if len(got) != 2 {
		t.Fatalf("missing = %+v, want outdated and absent", got)
	}
	if r := got[0]; r.DependentName != "a" || r.Reason != "outdated" || r.InstalledVersion != "1.0" {
		t.Fatalf("outdated row = %+v", r)
	}
	if r := got[1]; r.DependentName != "b" || r.Reason != "absent" {
		t.Fatalf("absent row = %+v", r)
	}
}
