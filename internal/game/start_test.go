package game

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/gamestore"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
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

// steamWithSMAPILine is a Steam whose MostRecent account's Stardew launch options run SMAPI from gameDir.
func steamWithSMAPILine(t *testing.T, gameDir string) *steam.Steam {
	t.Helper()
	root := t.TempDir()
	for rel, body := range map[string]string{
		"config/loginusers.vdf": `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`,
		"userdata/39734274/config/localconfig.vdf": `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "\"` +
			strings.ReplaceAll(filepath.Join(gameDir, "StardewModdingAPI.exe"), `\`, `\\`) + `\" %command%" } } } } } }`,
	} {
		path := filepath.Join(root, rel)
		testfs.WriteAt(t, path, []byte(body))
	}
	return &steam.Steam{Root: root}
}

func TestCommand(t *testing.T) {
	st := Starter{DataDir: t.TempDir(), FlatpakShow: func() (string, error) { return "", nil }}
	mods := filepath.FromSlash("/data/profiles/stardew/abc/mods")
	sm := &steam.Steam{Root: filepath.FromSlash("/steam")}
	fp := &steam.Steam{Root: filepath.FromSlash("/flatpak-steam"), Kind: steam.KindFlatpak}
	dir := filepath.FromSlash("/games/Stardew Valley")
	inst := Install{Game: "stardew", Dir: dir}
	withLine := steamWithSMAPILine(t, dir)
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
			StartEnv{Steam: withLine},
			"", "",
			[]string{filepath.Join(withLine.Root, "steam.exe"), "-applaunch", "413150", "--mods-path", mods},
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
	// flatpak override lists POSIX paths, so a drive letter's colon would read as a mode.
	if runtime.GOOS == "windows" {
		return
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

// Vortex and SMAPI's installer never set Steam's launch options, so a Windows Steam start would run the game without
// SMAPI; Mortar starts SMAPI itself until the options name it.
func TestWindowsStartsSMAPIDirectlyUntilSteamNamesIt(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		testfs.WriteAt(t, p, []byte(body))
	}
	write("config/loginusers.vdf", `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`)
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "-novid" } } } } } }`)
	plan := smapiPlan("windows", `C:\Stardew Valley`, filepath.Join(root, "mods"))
	env := StartEnv{Steam: &steam.Steam{Root: root}}
	inst := Install{Game: "stardew", Dir: `C:\Stardew Valley`}
	c, err := Starter{}.command("windows", inst, plan, env, "", "")
	if err != nil || c.Relay || !strings.EqualFold(filepath.Base(c.Name), "StardewModdingAPI.exe") {
		t.Fatalf("missing line: name = %q, relay = %v, err = %v", c.Name, c.Relay, err)
	}
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "\"C:\\Stardew Valley\\StardewModdingAPI.exe\" %command%" } } } } } }`)
	if c, _ = (Starter{}).command("windows", inst, plan, env, "", ""); !c.Relay || c.Failure != launch.HintSteam {
		t.Fatalf("line present: relay = %v, hint = %q", c.Relay, c.Failure)
	}
}

func TestStart(t *testing.T) {
	t.Parallel()
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
	if err := st.Start(ctx, inst, smapiPlan(runtime.GOOS, dir, mods), env, func(l []string) { lines = append(lines, l...) }); err != nil {
		t.Fatal(err)
	}
	// A Windows build starts the loader itself; Steam relays it elsewhere.
	launcher := "/usr/bin/steam"
	if runtime.GOOS == "windows" {
		launcher = filepath.Join(dir, "StardewModdingAPI.exe")
	}
	if ran[0] != launcher || len(lines) != 1 {
		t.Fatalf("ran %v, lines %q", ran, lines)
	}

	if runtime.GOOS == "windows" {
		return // the stale-log failure names Steam, which a direct Windows start does not use
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

func TestAModdedStartWithNoLogIsUpWhenTheProcessAppears(t *testing.T) {
	n := 0
	st := Starter{
		LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
		Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner:   func(string, string, ...string) (<-chan error, error) { return make(chan error), nil },
	}
	dir := t.TempDir()
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}, Ready: func() bool { n++; return n > 2 }}
	if err := st.Start(t.Context(), Install{Game: "stardew", Dir: dir}, smapiPlan("linux", dir, dir), env, nil); err != nil {
		t.Fatalf("a launch whose loader has no log waited for one: %v", err)
	}
}

func TestStartInFlatpakGoesThroughFlatpakSpawn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a Windows build starts the loader directly, and Flatpak is Linux-only")
	}
	t.Setenv("FLATPAK_ID", appversion.AppID)
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

func TestWindowsCommandKeepsArgumentsWholeAndInOrder(t *testing.T) {
	dir := `C:\Program Files (x86)\Steam\steamapps\common\Stardew Valley`
	mods := `C:\Users\O'Brien\AppData\Local\Mortar\profiles\stardew\a b\mods`
	extra := []string{"--log", `C:\Mortar Logs\say "hi".txt`, "--developer-mode"}
	plan := smapiPlan("windows", dir, mods, extra...)
	inst := Install{Game: "stardew", Dir: dir}
	direct, err := Starter{}.command("windows", inst, plan, StartEnv{Direct: true}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := append([]string{filepath.Join(dir, "StardewModdingAPI.exe"), "--mods-path", mods}, extra...)
	if got := append([]string{direct.Name}, direct.Args...); !slices.Equal(got, want) || direct.Dir != dir {
		t.Fatalf("direct = %q in %q, want %q", got, direct.Dir, want)
	}
	sm := steamWithSMAPILine(t, dir)
	relay, err := Starter{}.command("windows", inst, plan, StartEnv{Steam: sm}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Steam puts these after %command%, so the loader's own launch options stay ahead of the profile's.
	want = append([]string{filepath.Join(sm.Root, "steam.exe"), "-applaunch", "413150", "--mods-path", mods}, extra...)
	if got := append([]string{relay.Name}, relay.Args...); !slices.Equal(got, want) {
		t.Fatalf("relay = %q, want %q", got, want)
	}
}

func TestSteamLaunchOptionsKeepEnvironmentAheadOfTheLoader(t *testing.T) {
	exe := `C:\Program Files (x86)\Steam\steamapps\common\Stardew Valley\StardewModdingAPI.exe`
	quoted := `"` + exe + `"`
	env := `WINEDLLOVERRIDES="winhttp=n,b"`
	for _, tc := range []struct{ current, with string }{
		{env + " %command%", env + " " + quoted + " %command%"},
		{env + ` %command% --log "C:\a b\x.txt"`, env + " " + quoted + ` %command% --log "C:\a b\x.txt"`},
	} {
		got := launchWith(exe, tc.current)
		if got != tc.with {
			t.Errorf("with %q = %q, want %q", tc.current, got, tc.with)
		}
		if back := launchWithout(exe, got); back != tc.current {
			t.Errorf("without %q = %q, want %q", got, back, tc.current)
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

func TestOnlySteamAndEpicCopiesStartThroughTheirStoreAndLinuxRefusesAWindowsBuildWithNothingToRunIt(t *testing.T) {
	cases := []struct {
		inst        Install
		steam, runs bool
	}{
		{Install{Store: StoreSteam, Dir: "/g", Platform: "windows", Runtime: "proton"}, true, true},
		{Install{Store: StoreFlatpakSteam, Dir: "/g", Platform: "linux", Runtime: "native"}, true, true},
		{Install{Store: "", Dir: "/g", Platform: "linux", Runtime: "native"}, false, true},
		{Install{Store: StoreBottles, Dir: "/g", Prefix: "/b", Platform: "windows", Runtime: "wine-prefix"}, false, true},
		{Install{Store: "", Dir: "/g", Platform: "windows", Runtime: "native"}, false, runtime.GOOS == "windows"},
		{Install{Store: StoreLutris, Dir: "/g", Platform: "windows", Runtime: "native"}, false, runtime.GOOS == "windows"},
		{Install{Store: StoreEpic, Dir: "/g", Platform: "windows", Runtime: "native"}, true, runtime.GOOS == "windows"},
	}
	for _, c := range cases {
		if got := c.inst.StoreStarts(); got != c.steam {
			t.Errorf("%+v StoreStarts = %v", c.inst, got)
		}
		if err := c.inst.RunError("Game"); (err == nil) != c.runs {
			t.Errorf("%+v RunError = %v", c.inst, err)
		}
	}
}

func TestGuardedPutsTheWrapperFirst(t *testing.T) {
	name, args := guarded("", "game", []string{"-a"})
	if name != "game" || !slices.Equal(args, []string{"-a"}) {
		t.Fatalf("no wrapper: %s %v", name, args)
	}
	name, args = guarded("/guard", "game", []string{"-a"})
	if name != "/guard" || !slices.Equal(args, []string{"game", "-a"}) {
		t.Fatalf("wrapped: %s %v", name, args)
	}
}

// An Epic Games copy starts through its launcher's link, as a relay Mortar follows by the loader's log.
func TestEpicCopyStartsThroughTheEpicLauncher(t *testing.T) {
	inst := Install{Game: "enter-the-gungeon", Store: StoreEpic, Dir: filepath.FromSlash("/games/EtG")}
	c, err := Starter{}.command("windows", inst, &launchplan.Plan{Mode: launchplan.ModeProfile}, StartEnv{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"rundll32.exe", "url.dll,FileProtocolHandler", "com.epicgames.launcher://apps/Garlic?action=launch&silent=true"}
	if got := append([]string{c.Name}, c.Args...); !slices.Equal(got, want) || !c.Relay || c.Failure != launch.HintEpic {
		t.Fatalf("got %v relay %v hint %q, want %v", got, c.Relay, c.Failure, want)
	}
	for _, bad := range []string{"", "a&b", "x?y", "../x", "a b", "a/b"} {
		if components.PlainAppName(bad) {
			t.Fatalf("%q passed as a plain AppName", bad)
		}
	}
}

func TestDirectCommandStartsAFlatpakCopyThroughFlatpakAndALinuxBuildThroughItsStartFile(t *testing.T) {
	plan := &launchplan.Plan{Mode: launchplan.ModeProfile, Args: []string{"-o", "w"}, Env: map[string]string{"A": "1"}}
	c, err := Starter{}.command("linux", Install{Game: "vintage-story", Store: StoreFlatpak, Platform: "linux"}, plan, StartEnv{Direct: true}, "", "/usr/bin/flatpak")
	if err != nil || c.Name != "/usr/bin/flatpak" || !slices.Equal(c.Args, []string{"run", "--env=A=1", "at.vintagestory.VintageStory", "-o", "w"}) {
		t.Fatalf("flatpak = %+v, %v", c, err)
	}
	if _, err := (Starter{}).command("linux", Install{Game: "vintage-story", Store: StoreFlatpak, Platform: "linux"}, plan, StartEnv{Direct: true}, "", ""); err == nil {
		t.Fatal("a Flatpak copy started without flatpak")
	}
	dir := filepath.FromSlash("/games/vintagestory")
	c, err = Starter{}.Command("linux", Install{Game: "vintage-story", Store: gamestore.StoreStandalone, Dir: dir, Platform: "linux"}, plan)
	if err != nil || c.Name != filepath.Join(dir, "run.sh") {
		t.Fatalf("native = %+v, %v", c, err)
	}
}

func TestStartSchedulesDirectInstances(t *testing.T) {
	dir := t.TempDir()
	plan := vanillaPlan("linux", dir)
	for _, tc := range []struct {
		name         string
		direct       bool
		instances    int
		delay        int
		cancelSleep  bool
		testCopies   bool
		wantRoles    []string
		wantStarts   int
		wantDelays   []time.Duration
		wantCanceled bool
	}{
		{
			name:       "direct three with delay",
			direct:     true,
			instances:  3,
			delay:      5,
			wantStarts: 3,
			wantDelays: []time.Duration{5 * time.Second, 5 * time.Second},
		},
		{
			name:       "test copies of a loader with no ready line keep the delay",
			direct:     true,
			instances:  2,
			delay:      5,
			testCopies: true,
			wantStarts: 2,
			wantDelays: []time.Duration{5 * time.Second},
			wantRoles:  []string{"--lan --host --tile 1/2", "--lan --join 127.0.0.1 --tile 2/2"},
		},
		{
			name:       "steam url starts one",
			direct:     false,
			instances:  3,
			delay:      5,
			wantStarts: 1,
		},
		{
			name:         "cancel between instances",
			direct:       true,
			instances:    3,
			delay:        5,
			cancelSleep:  true,
			wantStarts:   1,
			wantDelays:   []time.Duration{5 * time.Second},
			wantCanceled: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var starts int
			var delays []time.Duration
			var roles []string
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			st := Starter{
				LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
				Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: time.Millisecond},
				Runner: func(_, _ string, args ...string) (<-chan error, error) {
					starts++
					if i := slices.Index(args, "--lan"); i >= 0 {
						roles = append(roles, strings.Join(args[i:], " "))
					}
					return make(chan error), nil
				},
				Sleep: func(_ context.Context, d time.Duration) error {
					delays = append(delays, d)
					if tc.cancelSleep {
						cancel()
						return context.Canceled
					}
					return nil
				},
			}
			env := StartEnv{
				Steam:        &steam.Steam{Root: t.TempDir()},
				Direct:       tc.direct,
				Instances:    tc.instances,
				DelaySeconds: tc.delay,
				TestCopies:   tc.testCopies,
				Ready:        func() bool { return true },
			}
			err := st.Start(ctx, Install{Game: "stardew", Dir: dir}, plan, env, nil)
			if tc.wantCanceled {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err = %v, want canceled", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if starts != tc.wantStarts || !slices.Equal(delays, tc.wantDelays) || !slices.Equal(roles, tc.wantRoles) {
				t.Fatalf("starts %d delays %v, want %d %v", starts, delays, tc.wantStarts, tc.wantDelays)
			}
		})
	}
}

// A loader's per-copy hook adjusts the plan of copies 2 and up only, on a clone of the plan.
func TestStartTestCopiesAppliesTheLoadersHookToLaterCopies(t *testing.T) {
	dir := t.TempDir()
	plan := vanillaPlan("linux", dir)
	plan.AddArgs("--target", "profile")
	plan.ForCopy = func(n int, c *launchplan.Plan) error {
		c.Args[slices.Index(c.Args, "--target")+1] = fmt.Sprintf("copy%d", n)
		return nil
	}
	var targets []string
	st := Starter{
		LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
		Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: time.Millisecond},
		Runner: func(_, _ string, args ...string) (<-chan error, error) {
			targets = append(targets, args[slices.Index(args, "--target")+1])
			return make(chan error), nil
		},
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}, Direct: true, Instances: 3, TestCopies: true, Ready: func() bool { return true }}
	if err := st.Start(t.Context(), Install{Game: "stardew", Dir: dir}, plan, env, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"profile", "copy2", "copy3"}; !slices.Equal(targets, want) {
		t.Fatalf("targets = %v, want %v", targets, want)
	}
	if slices.Contains(plan.Args, "copy2") {
		t.Fatalf("the shared plan changed: %v", plan.Args)
	}
}

// A test copy starts once the copy before it says in its log that the loader finished: copies share the game's save
// folder, and starting together they write the same files at once.
func TestStartTestCopiesWaitForTheLoaderBefore(t *testing.T) {
	for _, tc := range []struct {
		name       string
		log        string
		wantStarts int
	}{
		{"loaded", "[Message:   BepInEx] Chainloader startup complete\n", 2},
		{"never loaded", "[Info   :   BepInEx] Loading [A 1.0]\n", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			plan := vanillaPlan("linux", dir)
			plan.Mode = launchplan.ModeProfile
			logFile := filepath.Join(t.TempDir(), "LogOutput.log")
			starts := 0
			st := Starter{
				LookPath: func(string) (string, error) { return "/usr/bin/steam", nil },
				Timing:   launch.Timing{Timeout: 300 * time.Millisecond, Poll: time.Millisecond},
				Runner: func(string, string, ...string) (<-chan error, error) {
					starts++
					if starts == 1 {
						if err := os.WriteFile(logFile, []byte(tc.log), 0o600); err != nil {
							t.Error(err)
						}
					}
					return make(chan error), nil
				},
				Sleep: func(context.Context, time.Duration) error { t.Error("slept between test copies"); return nil },
			}
			env := StartEnv{
				Steam: &steam.Steam{Root: t.TempDir()}, Direct: true, Instances: 2, TestCopies: true, DelaySeconds: 5,
				LogFile: logFile, Ready: func() bool { return true },
				LoaderReady: func(line string) bool { return strings.Contains(line, "Chainloader startup complete") },
			}
			err := st.Start(t.Context(), Install{Game: "lethal-company", Dir: dir}, plan, env, nil)
			if starts != tc.wantStarts || (err == nil) != (tc.wantStarts == 2) {
				t.Fatalf("starts %d err %v, want %d", starts, err, tc.wantStarts)
			}
		})
	}
}

// Play starts the VR runtime the game is set to start first: SteamVR through Steam, ALVR's dashboard by its program.
func TestStartVRStartsTheRuntimeFirst(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Steam's command line is checked on Linux")
	}
	var ran [][]string
	st := Starter{
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Runner: func(_, name string, args ...string) (<-chan error, error) {
			ran = append(ran, append([]string{name}, args...))
			return make(chan error), nil
		},
	}
	env := StartEnv{Steam: &steam.Steam{Root: t.TempDir()}}
	for _, off := range []string{"", settings.StartFirstOff} {
		env.StartFirst = off
		if err := st.startVR(t.Context(), env); err != nil || len(ran) != 0 {
			t.Fatalf("%q: ran %v, %v", off, ran, err)
		}
	}
	env.StartFirst = settings.StartFirstSteamVR
	if err := st.startVR(t.Context(), env); err != nil || !slices.Equal(ran[0], []string{"/usr/bin/steam", "-applaunch", "250820"}) {
		t.Fatalf("SteamVR: ran %v, %v", ran, err)
	}
	env.StartFirst = settings.StartFirstALVR
	if err := st.startVR(t.Context(), env); err == nil {
		t.Fatal("ALVR with no program chosen started")
	}
	env.ALVRProgram = filepath.Join(t.TempDir(), "alvr-dashboard-not-running")
	if err := st.startVR(t.Context(), env); err != nil || len(ran) != 2 || ran[1][0] != env.ALVRProgram {
		t.Fatalf("ALVR: ran %v, %v", ran, err)
	}
}

func TestRunErrorSearchesAddedSteamFolders(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Proton is Linux-only")
	}
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	proton := filepath.Join(root, "steamapps", "common", "Proton 9.0")
	if err := os.MkdirAll(proton, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proton, "proton"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	in := Install{Dir: "/g", Platform: "windows", Runtime: "mortar-proton"}
	if in.RunError("Game") == nil {
		t.Fatal("no Proton anywhere must be an error")
	}
	if err := in.RunError("Game", root); err != nil {
		t.Fatalf("Proton in an added Steam folder: %v", err)
	}
}
