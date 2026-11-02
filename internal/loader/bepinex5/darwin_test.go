package bepinex5

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestContributeDarwin(t *testing.T) {
	g := components.GameInfo{ID: "g", Name: "G", DarwinMarker: "My Game.app/Contents/MacOS/My Game"}
	for name, files := range map[string][]string{
		"pack without the script": {"doorstop_libs/libdoorstop_x64.dylib"},
		"pack with the script":    {"run_bepinex.sh", "libdoorstop.dylib"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range files {
				path := filepath.Join(dir, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := safefs.WriteFile(path, []byte("executable_name=\"\"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			plan := launchplan.New(launchplan.ModeProfile)
			err := contributeDarwin(plan, loader.ProfileView{Dir: dir, Platform: "darwin"}, marker{Doorstop: 4}, g)
			if files[0] != "run_bepinex.sh" {
				if err == nil {
					t.Fatal("a pack without run_bepinex.sh started")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := `"` + filepath.Join(dir, "run_bepinex.sh") + `" %command%`
			if plan.SteamLaunchOption != want {
				t.Errorf("launch option %q, want %q", plan.SteamLaunchOption, want)
			}
			if plan.Exe != "" || len(plan.Files) != 0 {
				t.Errorf("darwin placed files or replaced the executable: %+v", plan)
			}
			for _, f := range files {
				if st, _ := os.Stat(filepath.Join(dir, f)); st.Mode().Perm() != 0o755 {
					t.Errorf("%s mode %v", f, st.Mode().Perm())
				}
			}
			if b, _ := safefs.ReadFile(filepath.Join(dir, "run_bepinex.sh")); !strings.Contains(string(b), "executable_name='My Game'") {
				t.Errorf("script %q", b)
			}
		})
	}
	if err := contributeDarwin(launchplan.New(launchplan.ModeProfile), loader.ProfileView{}, marker{}, components.GameInfo{Name: "G"}); err == nil {
		t.Error("a game without a macOS app started")
	}
}

func TestOwnedFilesCoverDarwinPack(t *testing.T) {
	got := strings.Join(Loader{}.ProfileFiles(), " ")
	for _, f := range []string{"run_bepinex.sh", "libdoorstop.dylib"} {
		if !strings.Contains(got, f) {
			t.Errorf("ProfileFiles lacks %s", f)
		}
	}
}
