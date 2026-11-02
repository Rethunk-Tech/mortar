//go:build !windows

package launchsvc

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const vsGame = "vintage-story"

// serverEnv is a Vintage Story install with one profile holding a mod, and a dotnet on the PATH.
func serverEnv(t *testing.T) (svc *Service, profileID, installDir string, inst game.Install) {
	t.Helper()
	useCatalog(t, func(*components.Manifest) {})
	bin := t.TempDir()
	testfs.WriteAt(t, filepath.Join(bin, "dotnet"), []byte("#!/bin/sh\n"))
	if err := safefs.Chmod(filepath.Join(bin, "dotnet"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	installDir = t.TempDir()
	testfs.WriteAt(t, filepath.Join(installDir, "Vintagestory"), []byte("x"))
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[vsGame] = installDir }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, vsGame, "A")
	dir, err := profiles.ProfileDir(vsGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteAt(t, filepath.Join(dir, "Mods", "mod", "modinfo.json"), []byte("{}"))
	svc = NewService(t.TempDir(), set, profiles)
	_, inst = svc.installs(game.Find(vsGame))
	inst.Platform = "linux"
	return svc, p.ID, installDir, inst
}

func TestAServerProfileStartsTheServerWithTheProfilesMods(t *testing.T) {
	svc, pid, installDir, inst := serverEnv(t)
	g := game.Find(vsGame)
	client, err := svc.planProfile(context.Background(), g, inst, pid, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if client.Exe != "" || len(client.RootDirs) != 0 || len(client.Args) != 0 {
		t.Fatalf("the client plan changed: %+v", client)
	}
	if _, err := svc.profiles.SetServerMode(vsGame, pid, true); err != nil {
		t.Fatal(err)
	}
	plan, err := svc.planProfile(context.Background(), g, inst, pid, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	data, err := svc.profiles.ServerDataDir(vsGame, pid)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(plan.Exe) != "dotnet" {
		t.Fatalf("exe = %q", plan.Exe)
	}
	if want := []string{filepath.Join(installDir, "VintagestoryServer.dll"), "--dataPath", data}; !slices.Equal(plan.Args, want) {
		t.Fatalf("args = %v, want %v", plan.Args, want)
	}
	if len(plan.Files) != len(client.Files) || len(plan.Files) == 0 {
		t.Fatalf("the server places %d files, the client %d", len(plan.Files), len(client.Files))
	}
	roots, err := svc.planRoots(vsGame, inst, plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := roots["mods"]; got != filepath.Join(data, "Mods") {
		t.Fatalf("mods root = %q, want the server's data folder", got)
	}
	if got, _ := svc.serverModConfig(vsGame, pid); got != filepath.Join(data, "ModConfig") {
		t.Fatalf("mod config = %q", got)
	}
}

func TestAServerRunsOnWindowsBuildsAsItsExecutable(t *testing.T) {
	svc, pid, installDir, inst := serverEnv(t)
	inst.Platform = "windows"
	if _, err := svc.profiles.SetServerMode(vsGame, pid, true); err != nil {
		t.Fatal(err)
	}
	plan, err := svc.planProfile(context.Background(), game.Find(vsGame), inst, pid, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Exe != filepath.Join(installDir, "VintagestoryServer.exe") || plan.Args[0] != "--dataPath" {
		t.Fatalf("exe %q args %v", plan.Exe, plan.Args)
	}
}

func TestTheServerProcessIsTheProfilesRunningGame(t *testing.T) {
	svc, _, installDir, inst := serverEnv(t)
	svc.procDir = t.TempDir()
	g := game.Find(vsGame)
	fakeProc(t, svc, "4242", "/usr/bin/dotnet", "/usr/bin/dotnet", filepath.Join(installDir, "VintagestoryServer.dll"), "--dataPath", "/x")
	fakeProc(t, svc, "4243", "/usr/bin/dotnet", "/usr/bin/dotnet", "/elsewhere/Other.dll")
	procs, err := svc.gameProcs(slot{g, inst.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) != 1 || procs[0].PID != 4242 {
		t.Fatalf("processes = %+v", procs)
	}
	if id, ok := svc.owner(g, procs[0]); !ok || id != inst.ID {
		t.Fatalf("owner = %q, %v", id, ok)
	}
}
