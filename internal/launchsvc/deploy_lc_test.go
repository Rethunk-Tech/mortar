//go:build !windows

package launchsvc

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[rel+"/"] = ""
		} else {
			b, _ := fsx.ReadFile(p)
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

func tsPackage(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	files["manifest.json"] = `{"name":"` + name + `","version_number":"1.0.0"}`
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), name+".zip"), files)
}

func TestALethalCompanyLaunchKeepsBepInExInTheProfile(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, ps := testenv.Stores(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "Friends")
	var keyA string
	for _, pkg := range []struct{ id, file string }{{"Ns-A", "A.dll"}, {"Ns-B", "B.dll"}} {
		zip := tsPackage(t, pkg.id[3:], map[string]string{"plugins/" + pkg.file: pkg.id, "config/" + pkg.id + ".cfg": "default " + pkg.id})
		res, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: pkg.id, Version: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.id == "Ns-A" {
			keyA = res.Profile.Entries[0].Key
		}
	}
	if err := ps.WriteFiles(lc, p.ID, map[string][]byte{"BepInEx/config/Ns-B.cfg": []byte("tuned Ns-B")}); err != nil {
		t.Fatal(err)
	}
	profileDir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, profileDir, "winhttp.dll", "proxy")
	testfs.WriteFile(t, profileDir, "doorstop_config.ini", "ini")

	install := t.TempDir()
	testfs.WriteFile(t, install, "Lethal Company.exe", "exe")
	testfs.WriteFile(t, install, "winhttp.dll", "the player's own")
	before := snapshotTree(t, install)
	inst := game.Install{ID: "lc1", Dir: install}
	svc := &Service{profiles: ps}

	launch := func() *deployment {
		plan := launchplan.New(launchplan.ModeProfile)
		for _, f := range []string{"winhttp.dll", "doorstop_config.ini"} {
			plan.AddFile(launchplan.PlanFile{Src: filepath.Join(profileDir, f), Dst: f})
		}
		if err := ps.SyncPackages(lc, p.ID); err != nil {
			t.Fatal(err)
		}
		dep, err := svc.deployProfile(t.Context(), lc, inst, p.ID, plan)
		if err != nil {
			t.Fatal(err)
		}
		return dep
	}

	cfg := filepath.Join(profileDir, "BepInEx", "config", "BepInEx.cfg")
	testfs.WriteFile(t, profileDir, "BepInEx/config/BepInEx.cfg", "[Logging.Console]\nEnabled = true\n")
	g, err := game.Require(lc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.launchPlan(t.Context(), g, inst, p.ID, launchplan.ModeProfile, "", "", ""); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(cfg); string(b) != "[Logging.Console]\nEnabled = true\n" {
		t.Fatalf("building the plan, as the launch preview does, rewrote BepInEx.cfg: %q", b)
	}

	dep := launch()
	if b, _ := fsx.ReadFile(cfg); string(b) != "[Logging.Console]\nEnabled = false\n" {
		t.Errorf("the launch left BepInEx's console window on: %q", b)
	}
	for rel, want := range map[string]string{
		"BepInEx/plugins/Ns-A/A.dll": "Ns-A",
		"BepInEx/plugins/Ns-B/B.dll": "Ns-B",
		"BepInEx/config/Ns-A.cfg":    "default Ns-A",
		"BepInEx/config/Ns-B.cfg":    "tuned Ns-B",
	} {
		if b, _ := fsx.ReadFile(filepath.Join(profileDir, rel)); string(b) != want {
			t.Errorf("profile %s = %q, want %q", rel, b, want)
		}
	}
	for rel, want := range map[string]string{"winhttp.dll": "proxy", "doorstop_config.ini": "ini"} {
		if b, _ := fsx.ReadFile(filepath.Join(install, rel)); string(b) != want {
			t.Errorf("install %s = %q, want %q", rel, b, want)
		}
	}
	if got := snapshotTree(t, install); len(got) != len(before)+1 {
		t.Errorf("the game folder holds more than the two doorstop files: %v", got)
	}

	// Steam patches the game while it runs; neither that nor anything else the game leaves is Mortar's to take.
	testfs.WriteFile(t, install, "Lethal Company.exe", "patched")
	testfs.WriteFile(t, install, "Lethal Company_Data/new.assets", "added")
	dep.unwind(t.Context())
	got := snapshotTree(t, install)
	if got["Lethal Company.exe"] != "patched" || got["Lethal Company_Data/new.assets"] != "added" || got["winhttp.dll"] != "the player's own" {
		t.Errorf("install after the launch: %v", got)
	}
	if _, ok := got["doorstop_config.ini"]; ok {
		t.Error("doorstop_config.ini outlives the launch")
	}
	if len(got) != len(before)+2 { // plus the added folder and file
		t.Errorf("install after the launch: %v, before %v", got, before)
	}

	if dep, err := svc.deployProfile(t.Context(), "stardew", inst, p.ID, launchplan.New(launchplan.ModeProfile)); err != nil || dep.d != nil {
		t.Fatalf("a redirected game deploys nothing: %v, %v", dep, err)
	}
	if _, err := ps.RemoveEntry(lc, p.ID, keyA); err != nil {
		t.Fatal(err)
	}
	launch().unwind(t.Context())
	if _, err := os.Lstat(filepath.Join(profileDir, "BepInEx", "plugins", "Ns-A")); err == nil {
		t.Error("a removed package's plugin stays in the profile")
	}
	if b, _ := fsx.ReadFile(filepath.Join(profileDir, "BepInEx", "plugins", "Ns-B", "B.dll")); string(b) != "Ns-B" {
		t.Errorf("the other package lost its plugin: %q", b)
	}
}
