package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestRelationsOfAThunderstorePackage(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	install := func(name, manifest string) string {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{"manifest.json": manifest, name + ".dll": "x"})
		res, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		return res.Profile.Entries[len(res.Profile.Entries)-1].Key
	}
	modKey := install("Mod", `{"name":"Mod","version_number":"1.0.0","dependencies":["BepInEx-BepInExPack-5.4.2100","Ns-Lib-1.0.0"]}`)
	libKey := install("Lib", `{"name":"Lib","version_number":"1.0.0","dependencies":[]}`)
	s := NewService(t.TempDir(), nil, profiles, nil)

	r, err := s.Relations("lethal-company", p.ID, modKey, "thunderstore:Ns-Mod")
	if err != nil {
		t.Fatal(err)
	}
	if r.PageURL != "https://thunderstore.io/c/lethal-company/p/Ns/Mod/" {
		t.Fatalf("page = %q, want the package's Thunderstore page", r.PageURL)
	}
	if len(r.Needs) != 1 || r.Needs[0].ID != mod.ID("thunderstore:Ns-Lib") || r.Needs[0].Name != "Lib" || r.Needs[0].State != "ok" {
		t.Fatalf("needs = %+v, want only Ns-Lib, met", r.Needs)
	}
	r, err = s.Relations("lethal-company", p.ID, libKey, "thunderstore:Ns-Lib")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.NeededBy) != 1 || r.NeededBy[0].ID != mod.ID("thunderstore:Ns-Mod") {
		t.Fatalf("neededBy = %+v, want Ns-Mod", r.NeededBy)
	}
}

func TestUpdatesAfterAProblemsCheckStillAskTheGameSources(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{"manifest.json": `{"name":"Mod","version_number":"1.0.0","dependencies":[]}`, "Mod.dll": "x"})
	if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore", items: []source.Item{{ID: "Ns-Mod", Name: "Mod", Author: "Ns", Version: "2.0.0"}}})
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
	if _, err := s.Problems(t.Context(), "lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.Updates(t.Context(), "lethal-company", p.ID)
	if err != nil || len(got.Updates) != 1 || got.Updates[0].Version != "2.0.0" {
		t.Fatalf("updates after a Problems check = %+v, %v", got.Updates, err)
	}
}

func TestEnabledPackagesCoverEverySourceForPluginClashes(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	install := func(src profile.Source) {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{"manifest.json": `{"name":"Mod","version_number":"1.0.0","dependencies":[]}`, "Mod.dll": "x"})
		if src.Kind == profile.KindGitHub {
			key, _, err := profiles.StageGitHub(t.Context(), "lethal-company", src, zip)
			if err == nil {
				_, err = profiles.InstallStaged("lethal-company", p.ID, key, src)
			}
			if err != nil {
				t.Fatal(err)
			}
			return
		}
		if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip, src); err != nil {
			t.Fatal(err)
		}
	}
	install(profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	install(profile.Source{Kind: profile.KindGitHub, Name: "mod.zip", Repo: "ns/mod", Tag: "1.0.0", Version: "1.0.0"})
	pkgs, err := profiles.EnabledPackages("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, pk := range pkgs {
		kinds[pk.Source] = true
	}
	if !kinds[profile.KindThunderstore] || !kinds[profile.KindGitHub] {
		t.Fatalf("packages = %+v, want the Thunderstore and the GitHub install", pkgs)
	}
}

func TestAGitHubInstallFindsItsThunderstoreTwinByPluginGUID(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	zip := func() string {
		return testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{"manifest.json": `{"name":"ConfigurableCompany","version_number":"3.6.0","dependencies":[]}`, "ConfigurableCompany.dll": "x"})
	}
	if _, err := profiles.InstallSource(t.Context(), "lethal-company", p.ID, zip(), profile.Source{Kind: profile.KindThunderstore, Name: "AMRV-ConfigurableCompany", Version: "3.6.0"}); err != nil {
		t.Fatal(err)
	}
	gh := profile.Source{Kind: profile.KindGitHub, Name: "github_release.zip", Repo: "TheAnsuz/Lethal-Company-Configurable-Company-API", Tag: "3.6.0", Version: "3.6.0"}
	key, _, err := profiles.StageGitHub(t.Context(), "lethal-company", gh, zip())
	if err == nil {
		_, err = profiles.InstallStaged("lethal-company", p.ID, key, gh)
	}
	if err != nil {
		t.Fatal(err)
	}
	pkgs, _ := profiles.EnabledPackages("lethal-company", p.ID)
	for _, pk := range pkgs {
		pluginMu.Lock()
		pluginCache[pluginKey{pk.Dir}] = dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: "com.amrv.configurablecompany", Version: "3.6.0"}}}
		pluginMu.Unlock()
	}
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore", items: []source.Item{{ID: "AMRV-ConfigurableCompany", Name: "ConfigurableCompany", Author: "AMRV", Version: "3.7.0"}}})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	s := NewService(t.TempDir(), nil, profiles, nil)
	ghMod := framework.Mod{Key: key, SourceKind: profile.KindGitHub, SourceRepo: gh.Repo, SourceVersion: "3.6.0"}
	ghMod.Name, ghMod.Version, ghMod.Author = "Amrv.ConfigurableCompany", "3.6.0", "Amrv"
	got := s.sourceUpdates(t.Context(), "lethal-company", []framework.Mod{ghMod}, nil)
	if len(got) != 1 || got[0].Key != key || got[0].Package != "AMRV-ConfigurableCompany" || !got[0].Switch || got[0].Version != "3.7.0" {
		t.Fatalf("updates = %+v, want the Thunderstore twin's 3.7.0", got)
	}
}
