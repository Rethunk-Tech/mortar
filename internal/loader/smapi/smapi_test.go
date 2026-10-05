package smapi

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestContributeAndVanilla(t *testing.T) {
	dir := filepath.FromSlash("/games/Stardew Valley")
	prof := loader.ProfileView{Dir: filepath.FromSlash("/data/profile"), InstallDir: dir}
	mods := filepath.Join(prof.Dir, "mods")

	linux := launchplan.New(launchplan.ModeProfile)
	if err := contribute("linux", linux, prof); err != nil {
		t.Fatal(err)
	}
	if want := []string{"--skip-terminal", "--", "--mods-path", mods}; !slices.Equal(linux.Args, want) || linux.Entry != filepath.Join(dir, "StardewValley") {
		t.Fatalf("linux plan = %q entry %q", linux.Args, linux.Entry)
	}
	win := launchplan.New(launchplan.ModeProfile)
	if err := contribute("windows", win, prof); err != nil {
		t.Fatal(err)
	}
	if want := []string{"--mods-path", mods}; !slices.Equal(win.Args, want) || win.Entry != filepath.Join(dir, "StardewModdingAPI.exe") {
		t.Fatalf("windows plan = %q entry %q", win.Args, win.Entry)
	}
	if err := contribute("linux", launchplan.New(launchplan.ModeProfile), loader.ProfileView{Dir: "profile"}); err == nil {
		t.Fatal("a relative mods folder must be refused")
	}

	van := launchplan.New(launchplan.ModeVanilla)
	if err := vanilla("linux", van, dir); err != nil || van.Exe != filepath.Join(dir, linuxOriginal) {
		t.Fatalf("linux vanilla exe = %q, %v", van.Exe, err)
	}
	van = launchplan.New(launchplan.ModeVanilla)
	if err := vanilla("windows", van, dir); err != nil || van.Exe != "" || van.Entry != filepath.Join(dir, "Stardew Valley.exe") {
		t.Fatalf("windows vanilla = %q, %q, %v", van.Exe, van.Entry, err)
	}
}

func TestOwnsAndReady(t *testing.T) {
	prof := loader.ProfileView{Dir: filepath.FromSlash("/data/profile")}
	mods := filepath.Join(prof.Dir, "mods")
	l := Loader{}
	if !l.Owns(loader.Process{Args: []string{"--mods-path", mods}}, prof) || l.Owns(loader.Process{Args: []string{"--mods-path", "/other"}}, prof) {
		t.Fatal("a process belongs to the profile whose mods folder it was started with")
	}
	if !l.Ready("[12:00:01 INFO  SMAPI] Loaded 2 mods") || l.Ready("[12:00:01 INFO  SMAPI] Starting") {
		t.Fatal("ready line")
	}
}

func TestGameVersion(t *testing.T) {
	const header = "SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix 6.16.8-200.fc42.x86_64"
	if got := (Loader{}).GameVersion(header + "\n[14:00:00 INFO SMAPI] Mods go here: /tmp/mods\n"); got != "1.6.15" {
		t.Fatalf("got %q", got)
	}
}
