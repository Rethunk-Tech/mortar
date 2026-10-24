package runtime

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// A Windows build Steam did not install runs through the best Proton a native Steam has, in Mortar's own prefix,
// where its paths resolve too.
func TestMortarProtonRunsAFolderCopyThroughSteamsBestProtonInMortarsPrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".local", "share", "Steam")
	for _, d := range []string{
		filepath.Join(root, "steamapps", "common", "Proton 8.0"),
		filepath.Join(root, "steamapps", "common", "Proton 9.0"),
		filepath.Join(root, "steamapps", "common", "Proton - Experimental"),
		filepath.Join(root, "compatibilitytools.d", "GE-Proton10-34"),
	} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "proton"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	inst := Install{Game: "x", Store: "", Dir: "/games/X", Platform: "windows", Home: home, GOOS: "linux"}
	if IDOf(inst) != MortarProton {
		t.Fatalf("runtime = %q", IDOf(inst))
	}
	if steamCopy := (Install{Game: "x", Store: storeSteam, AppID: "1", Dir: "/s/steamapps/common/X", Platform: "windows", GOOS: "linux"}); IDOf(steamCopy) != Proton {
		t.Fatal("a Steam copy stays on Steam's own Proton")
	}
	argv, env, err := ProtonCommand(inst, "/games/X/X.exe", "-a")
	if err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(home, "data", "mortar", "prefixes", "x")
	if want := []string{filepath.Join(root, "steamapps", "common", "Proton 9.0", "proton"), "waitforexitandrun", "/games/X/X.exe", "-a"}; !slices.Equal(argv, want) {
		t.Fatalf("argv = %v, want %v", argv, want)
	}
	if want := []string{"STEAM_COMPAT_DATA_PATH=" + prefix, "STEAM_COMPAT_CLIENT_INSTALL_PATH=" + root}; !slices.Equal(env, want) {
		t.Fatalf("env = %v, want %v", env, want)
	}
	inst.Proton = "GE-Proton10-34"
	if argv, _, err := ProtonCommand(inst, "/games/X/X.exe"); err != nil || argv[0] != filepath.Join(root, "compatibilitytools.d", "GE-Proton10-34", "proton") {
		t.Fatalf("a chosen build: %v %v", argv, err)
	}
	inst.Proton = "Proton 1.0"
	if argv, _, err := ProtonCommand(inst, "/games/X/X.exe"); err != nil || filepath.Base(filepath.Dir(argv[0])) != "Proton 9.0" {
		t.Fatalf("a build Steam no longer has falls back to the best: %v %v", argv, err)
	}
	got, err := Resolve(inst, components.PathTemplate{Windows: "{documents}/My Games/X"})
	if want := filepath.Join(prefix, "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "X"); err != nil || got != want {
		t.Fatalf("path = %q %v, want %q", got, err, want)
	}
}

// Builds outside Valve's numbered releases rank by the numbers in their names, so GE-Proton10 is newer than GE-Proton9.
func TestProtonRankOrdersOtherBuildsByTheirNumbers(t *testing.T) {
	got := []string{"GE-Proton9-27", "GE-Proton10-9", "GE-Proton10-10"}
	slices.SortFunc(got, protonRank)
	if want := []string{"GE-Proton10-10", "GE-Proton10-9", "GE-Proton9-27"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}
