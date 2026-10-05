//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestDeployPlacesLoaderFilesAndTakesThemBack(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	install, profileDir := t.TempDir(), t.TempDir()
	src := filepath.Join(profileDir, "winhttp.dll")
	if err := os.WriteFile(src, []byte("proxy"), 0o600); err != nil {
		t.Fatal(err)
	}
	inst := game.Install{ID: "abc", Dir: install}

	for _, plan := range []*launchplan.Plan{launchplan.New(launchplan.ModeProfile), launchplan.New(launchplan.ModeVanilla)} {
		plan.AddFile(launchplan.PlanFile{Src: src, Dst: "winhttp.dll"})
		dep, err := startDeploy(t.Context(), inst, plan, profile.DeployInputs{})
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
	if dep, err := startDeploy(t.Context(), inst, launchplan.New(launchplan.ModeProfile), profile.DeployInputs{}); dep.d != nil || err != nil {
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
