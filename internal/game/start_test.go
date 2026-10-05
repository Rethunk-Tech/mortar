package game

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// smapiPlan is the plan SMAPI's driver contributes for goos.
func smapiPlan(goos, dir, mods string, extra ...string) *launchplan.Plan {
	p := launchplan.New(launchplan.ModeProfile)
	if goos == "windows" {
		p.SetEntry(filepath.Join(dir, "StardewModdingAPI.exe"))
	} else {
		p.AddArgs("--skip-terminal", "--")
		p.SetEntry(filepath.Join(dir, "StardewValley"))
	}
	p.AddArgs("--mods-path", mods)
	p.AddArgs(extra...)
	return p
}

// vanillaPlan is the plan SMAPI's driver contributes for a vanilla launch.
func vanillaPlan(goos, dir string) *launchplan.Plan {
	p := launchplan.New(launchplan.ModeVanilla)
	if goos == "windows" {
		p.SetEntry(filepath.Join(dir, "Stardew Valley.exe"))
	} else {
		_ = p.OverrideExe("smapi", filepath.Join(dir, "StardewValley-original"))
	}
	return p
}

// injectedPlan is the plan an injecting loader (BepInEx) contributes: arguments only, no executable.
func injectedPlan(args ...string) *launchplan.Plan {
	p := launchplan.New(launchplan.ModeProfile)
	p.AddArgs(args...)
	return p
}

func TestCommand(t *testing.T) {
	st := Starter{DataDir: t.TempDir(), FlatpakShow: func() (string, error) { return "", nil }}
	mods := filepath.FromSlash("/data/profiles/stardew/abc/mods")
	sm := &steam.Steam{Root: filepath.FromSlash("/steam")}
	fp := &steam.Steam{Root: filepath.FromSlash("/flatpak-steam"), Kind: steam.KindFlatpak}
	dir := filepath.FromSlash("/games/Stardew Valley")
	inst := Install{Game: "stardew", Dir: dir}
	for _, tc := range []struct {
		name    string
		goos    string
		plan    *launchplan.Plan
		env     StartEnv
		steam   string
		flatpak string
		want    []string
		err     error
		hint    launch.Hint
	}{
		{
			"linux steam", "linux", smapiPlan("linux", dir, mods),
			StartEnv{Steam: sm},
			"/usr/bin/steam", "",
			[]string{"/usr/bin/steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods},
			nil, launch.HintSteam,
		},
		{
			"linux steam extra", "linux", smapiPlan("linux", dir, mods, "--developer-mode"),
			StartEnv{Steam: sm},
			"/usr/bin/steam", "",
			[]string{"/usr/bin/steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods, "--developer-mode"},
			nil, launch.HintSteam,
		},
		{
			"linux flatpak", "linux", smapiPlan("linux", dir, mods),
			StartEnv{Steam: fp},
			"", "/usr/bin/flatpak",
			[]string{"/usr/bin/flatpak", "run", "com.valvesoftware.Steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods},
			nil, launch.HintFlatpakFS,
		},
		{
			"windows steam", "windows", smapiPlan("windows", dir, mods),
			StartEnv{Steam: sm},
			"", "",
			[]string{filepath.Join(sm.Root, "steam.exe"), "-applaunch", "413150", "--mods-path", mods},
			nil, launch.HintSteam,
		},
		{
			"linux direct", "linux", smapiPlan("linux", dir, mods),
			StartEnv{Direct: true},
			"", "",
			[]string{filepath.Join(dir, "StardewValley"), "--skip-terminal", "--", "--mods-path", mods},
			nil, "",
		},
		{
			"windows direct", "windows", smapiPlan("windows", dir, mods),
			StartEnv{Direct: true},
			"", "",
			[]string{filepath.Join(dir, "StardewModdingAPI.exe"), "--mods-path", mods},
			nil, "",
		},
		{
			"linux steam vanilla", "linux", vanillaPlan("linux", dir),
			StartEnv{Steam: sm},
			"/usr/bin/steam", "",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil, "",
		},
		{
			"windows steam vanilla", "windows", vanillaPlan("windows", dir),
			StartEnv{Steam: sm},
			"", "",
			[]string{filepath.Join(sm.Root, "steam.exe"), "-applaunch", "413150"},
			nil, launch.HintSteam,
		},
		{
			"linux direct vanilla", "linux", vanillaPlan("linux", dir),
			StartEnv{Direct: true},
			"", "",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil, "",
		},
		{
			"windows direct vanilla", "windows", vanillaPlan("windows", dir),
			StartEnv{Direct: true},
			"", "",
			[]string{filepath.Join(dir, "Stardew Valley.exe")},
			nil, "",
		},
		{"no steam", "linux", smapiPlan("linux", dir, mods), StartEnv{}, "/usr/bin/steam", "", nil, launch.ErrNoSteam, ""},
		{"steam not on PATH", "linux", smapiPlan("linux", dir, mods), StartEnv{Steam: sm}, "", "", nil, launch.ErrNoSteam, ""},
		{"flatpak not on PATH", "linux", smapiPlan("linux", dir, mods), StartEnv{Steam: fp}, "", "", nil, launch.ErrNoSteam, ""},
		{
			"linux vanilla without steam", "linux", vanillaPlan("linux", dir),
			StartEnv{},
			"", "",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil, "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := st.command(tc.goos, inst, tc.plan, tc.env, tc.steam, tc.flatpak)
			if !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if tc.want != nil && !slices.Equal(append([]string{c.Name}, c.Args...), tc.want) {
				t.Fatalf("command = %v %v, want %v", c.Name, c.Args, tc.want)
			}
			if tc.want != nil && tc.hint != "" && c.Failure != tc.hint {
				t.Fatalf("hint = %q, want %q", c.Failure, tc.hint)
			}
		})
	}
	st.FlatpakShow = func() (string, error) { return "filesystems=" + st.DataDir + ":ro;", nil }
	c, err := st.command("linux", inst, smapiPlan("linux", dir, mods), StartEnv{Steam: fp}, "", "/usr/bin/flatpak")
	if err != nil || c.Failure != launch.HintSteam {
		t.Fatalf("granted override: hint = %q, err = %v", c.Failure, err)
	}
}

func TestDirectCommandIncludesPrefixAndEnvironment(t *testing.T) {
	dir := filepath.FromSlash("/games/Stardew Valley")
	plan := smapiPlan("linux", dir, filepath.FromSlash("/data/profile/mods"))
	plan.Prefix = []string{"gamemoderun", "mangohud"}
	plan.SetEnv("MORTAR_TEST", "1")
	cmd, err := Starter{}.Command("linux", Install{Game: "stardew", Dir: dir}, plan)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"mangohud", filepath.Join(dir, "StardewValley"), "--skip-terminal", "--", "--mods-path", filepath.FromSlash("/data/profile/mods")}
	if cmd.Name != "gamemoderun" || !slices.Equal(cmd.Args, want) || !slices.Equal(cmd.Env, []string{"MORTAR_TEST=1"}) {
		t.Fatalf("name = %q, args = %q, env = %q", cmd.Name, cmd.Args, cmd.Env)
	}
}

func TestDirectCommandStartsTheGameExecutableWhenTheLoaderNamesNone(t *testing.T) {
	dir := filepath.FromSlash("/games/Lethal Company")
	plan := injectedPlan("--doorstop-enabled", "true")
	cmd, err := Starter{}.Command("linux", Install{Game: "lethal-company", Dir: dir}, plan)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != filepath.Join(dir, "Lethal Company.exe") || !slices.Equal(cmd.Args, []string{"--doorstop-enabled", "true"}) {
		t.Fatalf("name = %q, args = %q", cmd.Name, cmd.Args)
	}
}

func TestWindowsHint(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("config/loginusers.vdf", `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`)
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "-novid" } } } } } }`)
	plan := smapiPlan("windows", `C:\Stardew Valley`, filepath.Join(root, "mods"))
	env := StartEnv{Steam: &steam.Steam{Root: root}}
	inst := Install{Game: "stardew"}
	c, err := Starter{}.command("windows", inst, plan, env, "", "")
	if err != nil || c.Failure != launch.HintLaunchOptions {
		t.Fatalf("missing line: hint = %q, err = %v", c.Failure, err)
	}
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "\"C:\\Stardew Valley\\StardewModdingAPI.exe\" %command%" } } } } } }`)
	if c, _ = (Starter{}).command("windows", inst, plan, env, "", ""); c.Failure != launch.HintSteam {
		t.Fatalf("line present: hint = %q", c.Failure)
	}
}

func TestStart(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "SMAPI-latest.txt")
	dir := t.TempDir()
	mods := filepath.Join(t.TempDir(), "mods")
	var ran []string
	st := Starter{
		LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
		Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner: func(_, name string, args ...string) (<-chan error, error) {
			ran = append([]string{name}, args...)
			return make(chan error), os.WriteFile(logFile, []byte("SMAPI 4.5.2 with Stardew Valley 1.6.15\n"), 0o600)
		},
	}
	var lines []string
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}, LogFile: logFile}
	inst := Install{Game: "stardew", Dir: dir}
	ctx := t.Context()
	if err := st.Start(ctx, inst, smapiPlan("linux", dir, mods), env, func(l []string) { lines = append(lines, l...) }); err != nil {
		t.Fatal(err)
	}
	if ran[0] != "/usr/bin/steam" || len(lines) != 1 {
		t.Fatalf("ran %v, lines %q", ran, lines)
	}

	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(logFile, old, old); err != nil {
		t.Fatal(err)
	}
	st.Runner = func(string, string, ...string) (<-chan error, error) { return make(chan error), nil }
	err := st.Start(ctx, inst, smapiPlan("linux", dir, mods), env, func([]string) {})
	var f *launch.Failure
	if !errors.As(err, &f) || f.Hint != launch.HintSteam {
		t.Fatalf("err = %v, want a steam-hint failure", err)
	}
}

func TestVanillaStartSucceedsWhenAProcessAppears(t *testing.T) {
	n := 0
	st := Starter{
		LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
		Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner:   func(string, string, ...string) (<-chan error, error) { return make(chan error), nil },
	}
	dir := t.TempDir()
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}, Ready: func() bool { n++; return n > 2 }}
	if err := st.Start(t.Context(), Install{Game: "stardew", Dir: dir}, vanillaPlan("linux", dir), env, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStartInFlatpakGoesThroughFlatpakSpawn(t *testing.T) {
	old := sandbox.Getenv
	sandbox.Getenv = func(k string) string {
		if k == "FLATPAK_ID" {
			return sandbox.AppID
		}
		return ""
	}
	t.Cleanup(func() { sandbox.Getenv = old })
	logFile := filepath.Join(t.TempDir(), "SMAPI-latest.txt")
	var ran []string
	st := Starter{
		LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
		Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner: func(dir, name string, args ...string) (<-chan error, error) {
			ran = append([]string{dir, name}, args...)
			return make(chan error), os.WriteFile(logFile, []byte("SMAPI 4.5.2\n"), 0o600)
		},
	}
	dir := t.TempDir()
	mods := filepath.Join(t.TempDir(), "mods")
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}, LogFile: logFile}
	if err := st.Start(t.Context(), Install{Game: "stardew", Dir: dir}, smapiPlan("linux", dir, mods), env, func([]string) {}); err != nil {
		t.Fatal(err)
	}
	want := []string{"", "flatpak-spawn", "--host", "/usr/bin/steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods}
	if !slices.Equal(ran[:len(want)], want) {
		t.Fatalf("ran %v, want prefix %v", ran, want)
	}
}

func TestSteamLaunchWithLoaderKeepsTheUsersOptions(t *testing.T) {
	exe := filepath.Join("games", "Stardew Valley", "StardewModdingAPI.exe")
	quoted := `"` + exe + `"`
	for current, want := range map[string]string{
		"":                        quoted + " %command%",
		"-windowed":               quoted + " %command% -windowed",
		"DXVK_HUD=1 %command% -x": "DXVK_HUD=1 " + quoted + " %command% -x",
		quoted + " %command%":     quoted + " %command%",
	} {
		if got := launchWith(exe, current); got != want {
			t.Errorf("%q -> %q, want %q", current, got, want)
		}
	}
}

func TestSteamLaunchWithoutLoaderKeepsTheUsersOptions(t *testing.T) {
	exe := "StardewModdingAPI.exe"
	quoted := `"` + filepath.Join("games", "Stardew Valley", exe) + `"`
	for current, want := range map[string]string{
		quoted + " %command%":                    "",
		"-windowed " + quoted + " %command%":     "-windowed %command%",
		"DXVK_HUD=1 " + quoted + " %command% -x": "DXVK_HUD=1 %command% -x",
		"%command%":                              "",
		"-windowed":                              "-windowed",
		"DXVK_HUD=1 %command% -x":                "DXVK_HUD=1 %command% -x",
	} {
		if got := launchWithout(exe, current); got != want {
			t.Errorf("%q -> %q, want %q", current, got, want)
		}
	}
}

func TestDirectCommandRunsBottleInstallThroughBottlesCLI(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Bottles is Linux only")
	}
	bottle := filepath.Join(t.TempDir(), "bottles", "Games")
	dir := filepath.Join(bottle, "drive_c", "Stardew Valley")
	mods := filepath.FromSlash("/data/profile/mods")
	plan := smapiPlan("windows", dir, mods)
	inst := Install{Game: "stardew", Store: StoreBottles, Dir: dir, Prefix: bottle, Platform: "windows", Runtime: "wine-prefix"}
	cmd, err := Starter{}.command("linux", inst, plan, StartEnv{Direct: true}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "-b", "Games", "-e", filepath.Join(dir, "StardewModdingAPI.exe"), "--", "--mods-path", mods}
	if cmd.Name != "bottles-cli" || !slices.Equal(cmd.Args, want) {
		t.Fatalf("command = %s %q", cmd.Name, cmd.Args)
	}
}
