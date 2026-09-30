package stardew

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

func TestCommand(t *testing.T) {
	g := Game{}
	mods := filepath.FromSlash("/data/profiles/stardew/abc/mods")
	st := &steam.Steam{Root: filepath.FromSlash("/steam")}
	dir := filepath.FromSlash("/games/Stardew Valley")
	for _, tc := range []struct {
		name  string
		goos  string
		req   launch.Request
		steam string
		want  []string
		err   error
	}{
		{
			"linux steam", "linux",
			launch.Request{ModsDir: mods, Steam: st},
			"/usr/bin/steam",
			[]string{"/usr/bin/steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods},
			nil,
		},
		{
			"linux steam extra", "linux",
			launch.Request{ModsDir: mods, Steam: st, ExtraArgs: []string{"--developer-mode"}},
			"/usr/bin/steam",
			[]string{"/usr/bin/steam", "-applaunch", "413150", "--skip-terminal", "--", "--mods-path", mods, "--developer-mode"},
			nil,
		},
		{
			"windows steam", "windows",
			launch.Request{ModsDir: mods, Steam: st},
			"",
			[]string{filepath.Join(st.Root, "steam.exe"), "-applaunch", "413150", "--mods-path", mods},
			nil,
		},
		{
			"windows steam extra", "windows",
			launch.Request{ModsDir: mods, Steam: st, ExtraArgs: []string{"--developer-mode"}},
			"",
			[]string{filepath.Join(st.Root, "steam.exe"), "-applaunch", "413150", "--mods-path", mods, "--developer-mode"},
			nil,
		},
		{
			"linux direct", "linux",
			launch.Request{ModsDir: mods, InstallDir: dir, Direct: true},
			"",
			[]string{filepath.Join(dir, "StardewValley"), "--skip-terminal", "--", "--mods-path", mods},
			nil,
		},
		{
			"windows direct", "windows",
			launch.Request{ModsDir: mods, InstallDir: dir, Direct: true},
			"",
			[]string{filepath.Join(dir, "StardewModdingAPI.exe"), "--mods-path", mods},
			nil,
		},
		{
			"linux steam vanilla", "linux",
			launch.Request{Vanilla: true, Steam: st, InstallDir: dir},
			"/usr/bin/steam",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil,
		},
		{
			"windows steam vanilla", "windows",
			launch.Request{Vanilla: true, Steam: st},
			"",
			[]string{filepath.Join(st.Root, "steam.exe"), "-applaunch", "413150"},
			nil,
		},
		{
			"linux direct vanilla", "linux",
			launch.Request{Vanilla: true, InstallDir: dir, Direct: true},
			"",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil,
		},
		{
			"windows direct vanilla", "windows",
			launch.Request{Vanilla: true, InstallDir: dir, Direct: true},
			"",
			[]string{filepath.Join(dir, "Stardew Valley.exe")},
			nil,
		},
		{"no steam", "linux", launch.Request{ModsDir: mods}, "/usr/bin/steam", nil, launch.ErrNoSteam},
		{"steam not on PATH", "linux", launch.Request{ModsDir: mods, Steam: st}, "", nil, launch.ErrNoSteam},
		{
			"linux vanilla without steam", "linux",
			launch.Request{Vanilla: true, InstallDir: dir},
			"",
			[]string{filepath.Join(dir, "StardewValley-original")},
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := g.command(tc.goos, tc.req, tc.steam)
			if !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if tc.want != nil && !slices.Equal(append([]string{c.Name}, c.Args...), tc.want) {
				t.Fatalf("command = %v %v, want %v", c.Name, c.Args, tc.want)
			}
		})
	}
	if _, err := g.command("linux", launch.Request{ModsDir: "mods", Steam: st}, "/usr/bin/steam"); err == nil {
		t.Fatal("a relative mods path must be refused")
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
	req := launch.Request{ModsDir: filepath.Join(root, "mods"), Steam: &steam.Steam{Root: root}}
	if abs, _ := filepath.Abs(req.ModsDir); !filepath.IsAbs(abs) {
		t.Skip("no absolute path")
	}
	c, err := Game{}.command("windows", req, "")
	if err != nil || c.Failure != launch.HintLaunchOptions {
		t.Fatalf("missing line: hint = %q, err = %v", c.Failure, err)
	}
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "\"C:\\Stardew Valley\\StardewModdingAPI.exe\" %command%" } } } } } }`)
	if c, _ = (Game{}).command("windows", req, ""); c.Failure != launch.HintSteam {
		t.Fatalf("line present: hint = %q", c.Failure)
	}
}

func TestLaunch(t *testing.T) {
	logDir := t.TempDir()
	mods := filepath.Join(t.TempDir(), "mods")
	var ran []string
	g := Game{
		LogDir:       logDir,
		LookPath:     func(string) (string, error) { return "/usr/bin/steam", nil },
		LaunchTiming: launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner: func(_, name string, args ...string) (<-chan error, error) {
			ran = append([]string{name}, args...)
			return make(chan error), os.WriteFile(filepath.Join(logDir, "SMAPI-latest.txt"), []byte("SMAPI 4.5.2 with Stardew Valley 1.6.15\n"), 0o600)
		},
	}
	var lines []string
	req := launch.Request{ModsDir: mods, Steam: &steam.Steam{Root: t.TempDir()}}
	ctx, cancel := context.WithCancel(context.Background())
	if err := g.Launch(ctx, req, func(l []string) { lines = append(lines, l...) }); err != nil {
		t.Fatal(err)
	}
	cancel()
	if ran[0] != "/usr/bin/steam" || len(lines) != 1 {
		t.Fatalf("ran %v, lines %q", ran, lines)
	}

	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(logDir, "SMAPI-latest.txt"), old, old); err != nil {
		t.Fatal(err)
	}
	g.Runner = func(string, string, ...string) (<-chan error, error) { return make(chan error), nil }
	err := g.Launch(context.Background(), req, func([]string) {})
	var f *launch.Failure
	if !errors.As(err, &f) || f.Hint != launch.HintSteam {
		t.Fatalf("err = %v, want a steam-hint failure", err)
	}
}

func TestVanillaLaunchSucceedsWhenAProcessAppears(t *testing.T) {
	n := 0
	g := Game{
		LookPath:     func(string) (string, error) { return "/usr/bin/steam", nil },
		LaunchTiming: launch.Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond},
		Runner: func(string, string, ...string) (<-chan error, error) {
			return make(chan error), nil
		},
	}
	req := launch.Request{
		Vanilla:    true,
		InstallDir: t.TempDir(),
		Steam:      &steam.Steam{Root: t.TempDir()},
		Seen: func() bool {
			n++
			return n > 2
		},
	}
	if err := g.Launch(t.Context(), req, nil); err != nil {
		t.Fatal(err)
	}
}
