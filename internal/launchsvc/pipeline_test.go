//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

func TestDeployPlacesLoaderFilesAndTakesThemBack(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	install, profileDir := t.TempDir(), t.TempDir()
	src := filepath.Join(profileDir, "winhttp.dll")
	if err := os.WriteFile(src, []byte("proxy"), 0o600); err != nil {
		t.Fatal(err)
	}
	inst := game.Install{ID: "abc", Dir: install}

	for _, plan := range []*launchplan.Plan{launchplan.New(launchplan.ModeProfile), launchplan.New(launchplan.ModeVanilla)} {
		plan.AddFile(launchplan.PlanFile{Src: src, Dst: "winhttp.dll"})
		dep, err := startDeploy(t.Context(), inst, plan)
		if err != nil || (plan.Mode == launchplan.ModeVanilla) != (dep.d == nil) {
			t.Fatalf("mode %s: deployment %v, %v", plan.Mode, dep, err)
		}
		if plan.Mode == launchplan.ModeProfile {
			if _, err := os.Lstat(filepath.Join(install, "winhttp.dll")); err != nil {
				t.Fatalf("the loader's file is not in the install: %v", err)
			}
		}
		dep.unwind(t.Context())
		dep.unwind(t.Context())
		if _, err := os.Lstat(filepath.Join(install, "winhttp.dll")); !os.IsNotExist(err) {
			t.Fatalf("mode %s: the placed file outlives the launch: %v", plan.Mode, err)
		}
	}
	if dep, err := startDeploy(t.Context(), inst, launchplan.New(launchplan.ModeProfile)); dep.d != nil || err != nil {
		t.Fatalf("a redirect loader declares no files, so nothing deploys: %v, %v", dep, err)
	}
}

func TestEnsureRuntimeEditsAnExistingProtonPrefix(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "steamapps", "common", "Lethal Company")
	reg := filepath.Join(home, "steamapps", "compatdata", "1966720", "pfx", "user.reg")
	svc := &Service{home: home}
	inst := game.Install{Game: "lethal-company", Store: "steam", Dir: dir, Platform: "windows"}
	reqs := []launchplan.RuntimeReq{{Kind: "dll-override", Key: "winhttp", Value: "native,builtin"}}
	if err := svc.ensureRuntime(inst, reqs); err != nil {
		t.Fatalf("a prefix Steam has not made yet is left alone: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(reg), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reg, []byte("WINE REGISTRY Version 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.ensureRuntime(inst, reqs); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(reg); !strings.Contains(string(b), "winhttp") {
		t.Fatalf("user.reg = %q", b)
	}
}

func TestDeployRecoversALaunchNothingTookBack(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	install := t.TempDir()
	src := filepath.Join(t.TempDir(), "winhttp.dll")
	for path, body := range map[string]string{src: "proxy", filepath.Join(install, "winhttp.dll"): "the player's own"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	inst := game.Install{ID: "abc", Dir: install}
	launch := func() *deployment {
		plan := launchplan.New(launchplan.ModeProfile)
		plan.AddFile(launchplan.PlanFile{Src: src, Dst: "winhttp.dll"})
		dep, err := startDeploy(t.Context(), inst, plan)
		if err != nil {
			t.Fatal(err)
		}
		return dep
	}
	launch() // Mortar lost this one
	launch().unwind(t.Context())
	if b, _ := fsx.ReadFile(filepath.Join(install, "winhttp.dll")); string(b) != "the player's own" {
		t.Fatalf("the player's file is now %q", b)
	}
}
