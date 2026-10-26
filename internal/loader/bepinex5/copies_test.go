package bepinex5

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildCopyRootKeepsTheCopysOwnConfig(t *testing.T) {
	profile, root := t.TempDir(), filepath.Join(t.TempDir(), "2")
	write(t, filepath.Join(profile, "BepInEx", "core", "BepInEx.Preloader.dll"), "dll")
	write(t, filepath.Join(profile, "BepInEx", "plugins", "a.dll"), "a")
	write(t, filepath.Join(profile, "BepInEx", "config", "x.cfg"), "host")
	write(t, filepath.Join(profile, "BepInEx", "LogOutput.log"), "host log")
	write(t, filepath.Join(profile, "BepInEx", "LogOutput.log.1"), "host log 1")
	write(t, filepath.Join(profile, "BepInEx", "changelog.txt"), "v1")
	if err := BuildCopyRoot(profile, root); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(root, "BepInEx")
	if got := read(t, filepath.Join(own, "core", "BepInEx.Preloader.dll")); got != "dll" {
		t.Fatalf("core through the link = %q", got)
	}
	if fi, err := os.Lstat(filepath.Join(own, "plugins")); err != nil || fi.Mode().IsRegular() {
		t.Fatalf("plugins is not a link: %v %v", fi, err)
	}
	if got := read(t, filepath.Join(own, "changelog.txt")); got != "v1" {
		t.Fatalf("file = %q", got)
	}
	if got := read(t, filepath.Join(own, "config", "x.cfg")); got != "host" {
		t.Fatalf("seeded config = %q", got)
	}
	for _, name := range []string{"LogOutput.log", "LogOutput.log.1"} {
		if _, err := os.Stat(filepath.Join(own, name)); err == nil {
			t.Fatalf("%s was copied", name)
		}
	}
	// The copy's edit survives a rebuild, the profile's files are refreshed.
	write(t, filepath.Join(own, "config", "x.cfg"), "joiner")
	write(t, filepath.Join(profile, "BepInEx", "config", "x.cfg"), "host2")
	write(t, filepath.Join(profile, "BepInEx", "changelog.txt"), "v2")
	if err := BuildCopyRoot(profile, root); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(own, "config", "x.cfg")); got != "joiner" {
		t.Fatalf("config after rebuild = %q", got)
	}
	if got := read(t, filepath.Join(own, "changelog.txt")); got != "v2" {
		t.Fatalf("file after rebuild = %q", got)
	}
	// Replacing the link must not reach through it into the profile.
	if got := read(t, filepath.Join(profile, "BepInEx", "plugins", "a.dll")); got != "a" {
		t.Fatalf("the profile's plugin after rebuild = %q", got)
	}
}

func TestForCopyRootsTheCopyAndOwnsIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	profiles := t.TempDir()
	mine, other := filepath.Join(profiles, "mine"), filepath.Join(profiles, "other")
	for _, p := range []string{mine, other} {
		write(t, filepath.Join(p, "BepInEx", "core", "BepInEx.Preloader.dll"), "dll")
	}
	view := loader.ProfileView{Game: "valheim", Dir: mine, Runtime: "native"}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), plan, view); err != nil {
		t.Fatal(err)
	}
	first := slices.Clone(plan.Args)
	copied := *plan
	copied.Args, copied.Env = slices.Clone(plan.Args), maps.Clone(plan.Env)
	if err := plan.ForCopy(2, &copied); err != nil {
		t.Fatal(err)
	}
	i := slices.Index(copied.Args, "--doorstop-target")
	if i < 0 {
		i = slices.Index(copied.Args, "--doorstop-target-assembly")
	}
	target := copied.Args[i+1]
	if !strings.Contains(target, filepath.Join("test-copies", "valheim", "mine", "2", "BepInEx", "core")) {
		t.Fatalf("copy target = %q", target)
	}
	if !slices.Equal(plan.Args, first) {
		t.Fatalf("the first copy's args changed: %v", plan.Args)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("copy target is not there: %v", err)
	}
	proc := loader.Process{Args: []string{"game", "--doorstop-target", target}}
	if !(Loader{}).Owns(proc, view) {
		t.Fatal("the profile does not own its copy's process")
	}
	if (Loader{}).Owns(proc, loader.ProfileView{Game: "valheim", Dir: other}) {
		t.Fatal("another profile owns the copy's process")
	}
}
