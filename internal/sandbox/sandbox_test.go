package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func index(a []string, seq ...string) int {
	for i := range a {
		if i+len(seq) <= len(a) && slices.Equal(a[i:i+len(seq)], seq) {
			return i
		}
	}
	return -1
}

func TestArgvHidesHomeThenBindsWhatTheGameNeeds(t *testing.T) {
	root := t.TempDir()
	home, game, saves, ro := filepath.Join(root, "home"), filepath.Join(root, "game"), filepath.Join(root, "home", ".config", "Saves"), filepath.Join(root, "steam")
	for _, d := range []string{home, game, saves, ro} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	a := Argv("/usr/bin/bwrap", Spec{Home: home, RW: []string{game, saves}, RO: []string{ro, filepath.Join(root, "absent")}}, "/g/run", []string{"-x"})

	hide := index(a, "--tmpfs", home)
	if hide < 0 {
		t.Fatalf("home not hidden: %v", a)
	}
	for _, bind := range [][]string{{"--bind", saves, saves}, {"--ro-bind", ro, ro}, {"--bind", game, game}} {
		if i := index(a, bind...); i < hide {
			t.Fatalf("%v at %d must follow the home tmpfs at %d: %v", bind, i, hide, a)
		}
	}
	if index(a, "--ro-bind", filepath.Join(root, "absent")) >= 0 {
		t.Fatalf("a missing read-only folder was bound: %v", a)
	}
	if index(a, "--unshare-net") >= 0 || index(a, "--unshare-pid") >= 0 {
		t.Fatalf("network and pid namespaces must stay shared: %v", a)
	}
	if !slices.Equal(a[len(a)-3:], []string{"--", "/g/run", "-x"}) {
		t.Fatalf("command not last: %v", a)
	}
	if a[0] != "/usr/bin/bwrap" || index(a, "--die-with-parent") < 0 || index(a, "--ro-bind", "/", "/") < 0 {
		t.Fatalf("base flags missing: %v", a)
	}
}

func TestArgvKeepsOnlyNamedSocketsInTheRuntimeDir(t *testing.T) {
	run := t.TempDir()
	wl := filepath.Join(run, "wayland-0")
	if err := os.WriteFile(wl, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a := Argv("bwrap", Spec{Runtime: run, Sockets: []string{wl, filepath.Join(run, "pipewire-0")}}, "g", nil)
	if i, j := index(a, "--tmpfs", run), index(a, "--bind", wl, wl); i < 0 || j < i {
		t.Fatalf("runtime dir must be emptied before the socket is bound: %v", a)
	}
	if index(a, "--bind", filepath.Join(run, "pipewire-0")) >= 0 {
		t.Fatalf("an absent socket was bound: %v", a)
	}
}

func TestArgvCacheAndCredentials(t *testing.T) {
	cache := t.TempDir()
	a := Argv("bwrap", Spec{CacheDir: cache, Environ: []string{"HOME=/h", "GITHUB_TOKEN=x", "SSH_AUTH_SOCK=/s", "PATH=/bin", "DBUS_SESSION_BUS_ADDRESS=u"}}, "g", nil)
	if index(a, "--setenv", "MESA_SHADER_CACHE_DIR", cache) < 0 || index(a, "--bind", cache, cache) < 0 {
		t.Fatalf("cache not bound: %v", a)
	}
	for _, k := range []string{"GITHUB_TOKEN", "SSH_AUTH_SOCK", "DBUS_SESSION_BUS_ADDRESS"} {
		if index(a, "--unsetenv", k) < 0 {
			t.Fatalf("%s kept: %v", k, a)
		}
	}
	if strings.Contains(strings.Join(a, " "), "--unsetenv PATH") || strings.Contains(strings.Join(a, " "), "--unsetenv HOME") {
		t.Fatalf("a harmless variable was removed: %v", a)
	}
}

func TestSteamROLeavesOutLoginAndConfig(t *testing.T) {
	for _, p := range SteamRO("/s") {
		if strings.Contains(p, "config") || strings.Contains(p, "userdata") || strings.Contains(p, "compatdata") {
			t.Fatalf("%s must stay hidden", p)
		}
	}
}

func TestArgvKeepsMissingRWStrict(t *testing.T) {
	a := Argv("bwrap", Spec{RW: []string{"/definitely/not/here"}}, "g", nil)
	if index(a, "--bind", "/definitely/not/here", "/definitely/not/here") < 0 {
		t.Fatalf("a missing writable folder must reach bwrap, which fails the launch: %v", a)
	}
}

func TestDenialsFindsOnlyPathsTheSandboxHides(t *testing.T) {
	log := strings.Join([]string{
		"[Error] UnauthorizedAccessException: Access to the path '/home/u/.ssh/id_rsa' is denied.",
		"[Error] Could not find a part of the path 'Z:\\home\\u\\.config\\mod\\a.json'.",
		"[Warning] No such file or directory: /mnt/games/Steam/steamapps/common/G/missing.dll",
		"[Error] Permission denied: /mnt/data/x",
		"[Error] Could not find a part of the path '/home/u/.config/StardewValley/Saves/x'.",
		"[Info] loaded /home/u/.ssh/id_rsa",
		"[Error] Permission denied: /home/u/.ssh/id_rsa",
	}, "\n")
	got := Denials([]string{log}, []string{"/home/u/.config/StardewValley"}, "/home/u")
	want := []string{"/home/u/.ssh/id_rsa", "/home/u/.config/mod/a.json", "/mnt/data/x"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
