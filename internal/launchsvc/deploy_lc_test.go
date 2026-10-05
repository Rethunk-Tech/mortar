//go:build !windows

package launchsvc

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
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

func TestDeployIntoALethalCompanyInstallAndBack(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "Friends")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []struct{ id, file string }{{"Ns-A", "A.dll"}, {"Ns-B", "B.dll"}} {
		zip := tsPackage(t, pkg.id[3:], map[string]string{"plugins/" + pkg.file: pkg.id, "config/" + pkg.id + ".cfg": "default " + pkg.id})
		if _, err := ps.InstallSource(lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: pkg.id, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := ps.WriteFiles(lc, p.ID, map[string][]byte{"BepInEx/config/Ns-B.cfg": []byte("tuned Ns-B")}); err != nil {
		t.Fatal(err)
	}
	profileDir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}

	install := t.TempDir()
	testfs.WriteFile(t, install, "Lethal Company.exe", "exe")
	testfs.WriteFile(t, install, "BepInEx/config/user.cfg", "the player's own")
	before := snapshotTree(t, install)
	inst := game.Install{ID: "lc1", Dir: install}
	svc := &Service{profiles: ps}

	launch := func() *deployment {
		plan := launchplan.New(launchplan.ModeProfile)
		plan.AddFile(launchplan.PlanFile{Src: filepath.Join(profileDir, "winhttp.dll"), Dst: "winhttp.dll"})
		dep, err := svc.deployProfile(t.Context(), lc, inst, p.ID, plan)
		if err != nil {
			t.Fatal(err)
		}
		return dep
	}
	testfs.WriteFile(t, profileDir, "winhttp.dll", "proxy")

	dep := launch()
	for rel, want := range map[string]string{
		"winhttp.dll":                "proxy",
		"BepInEx/plugins/Ns-A/A.dll": "Ns-A",
		"BepInEx/plugins/Ns-B/B.dll": "Ns-B",
		"BepInEx/config/Ns-A.cfg":    "default Ns-A",
		"BepInEx/config/Ns-B.cfg":    "tuned Ns-B",
		"BepInEx/config/user.cfg":    "the player's own",
	} {
		if b, _ := fsx.ReadFile(filepath.Join(install, rel)); string(b) != want {
			t.Errorf("%s = %q, want %q", rel, b, want)
		}
	}
	// The game runs: it writes a config of its own, edits a placed one, and leaves a stray file.
	testfs.WriteFile(t, install, "BepInEx/config/new.cfg", "generated")
	testfs.WriteFile(t, install, "BepInEx/config/Ns-A.cfg", "edited in game")
	testfs.WriteFile(t, install, "stray.log", "log")
	dep.unwind(t.Context())

	if got := snapshotTree(t, install); len(got) != len(before) {
		t.Fatalf("install after the launch\n got %v\nwant %v", got, before)
	}
	for k, v := range before {
		if snapshotTree(t, install)[k] != v {
			t.Errorf("%s is not as it was", k)
		}
	}
	for rel, want := range map[string]string{
		"BepInEx/config/new.cfg":      "generated",
		"BepInEx/config/Ns-A.cfg":     "edited in game",
		"overwrite/profile/stray.log": "log",
	} {
		if b, _ := fsx.ReadFile(filepath.Join(profileDir, rel)); string(b) != want {
			t.Errorf("profile %s = %q, want %q", rel, b, want)
		}
	}

	if dep, err := svc.deployProfile(t.Context(), "stardew", inst, p.ID, launchplan.New(launchplan.ModeProfile)); err != nil || dep.d != nil {
		t.Fatalf("a redirected game deploys nothing: %v, %v", dep, err)
	}
	dep = launch()
	if b, _ := fsx.ReadFile(filepath.Join(install, "stray.log")); string(b) != "log" {
		t.Errorf("the overwrite folder is not deployed on the next launch: %q", b)
	}
	dep.unwind(t.Context())
	if _, err := os.Lstat(filepath.Join(install, "stray.log")); err == nil {
		t.Error("the overwrite file outlives the launch")
	}
}
