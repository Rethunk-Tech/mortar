package nxm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

// flatpakLinux is newLinux inside a Flatpak whose host is this machine: a host command runs here, against the test's
// folders, and a host gio only records what it was asked to launch.
func flatpakLinux(t *testing.T, current string) (*System, string) {
	t.Helper()
	packaged = "flatpak"
	t.Cleanup(func() { packaged = "" })
	l, r := newLinux(t, current)
	bin := t.TempDir()
	launched := filepath.Join(bin, "launched")
	gio := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + launched + "\n"
	if err := fsx.WriteFile(filepath.Join(bin, "gio"), []byte(gio), 0o700); err != nil {
		t.Fatal(err)
	}
	l.run = func(name string, args ...string) (string, error) {
		if name != "flatpak-spawn" || len(args) < 2 || args[0] != "--host" || args[1] == "xdg-mime" {
			return r.run(name, args...)
		}
		r.calls = append(r.calls, name+" "+strings.Join(args, " "))
		if args[1] != "sh" {
			t.Fatalf("host command %q: the bridge only runs sh scripts and xdg-mime", args[1])
		}
		return execRun("sh", args[2:]...)
	}
	t.Setenv("HOME", l.home)
	t.Setenv("XDG_CONFIG_HOME", hostConfig(l))
	t.Setenv("XDG_DATA_HOME", l.dataHome)
	t.Setenv("XDG_DATA_DIRS", t.TempDir())
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	return l, launched
}

// hostConfig is the host's config folder in flatpakLinux, apart from the sandbox's own l.configHome.
func hostConfig(l *System) string { return filepath.Join(l.home, "host-config") }

func TestFlatpakRestoreDropsOurDefaultFromTheHostsMimeapps(t *testing.T) {
	l, _ := flatpakLinux(t, "")
	if err := os.MkdirAll(hostConfig(l), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(hostConfig(l), "dotfiles.list")
	list := filepath.Join(hostConfig(l), "mimeapps.list")
	if err := fsx.WriteFile(target, []byte("[Default Applications]\nx-scheme-handler/nxm="+desktopID+";\ntext/html=a.desktop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, list); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(nil); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(list); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("mimeapps.list symlink replaced: %v", err)
	}
	if b, _ := fsx.ReadFile(target); string(b) != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("host mimeapps.list: %q", b)
	}
	if _, err := os.Stat(filepath.Join(l.configHome, "mimeapps.list")); !os.IsNotExist(err) {
		t.Errorf("wrote the sandbox's mimeapps.list: %v", err)
	}
}

func TestFlatpakForwardsToThePreviousHandlerOnTheHost(t *testing.T) {
	l, launched := flatpakLinux(t, "")
	entry := filepath.Join(l.dataHome, "applications", "other.desktop")
	if err := os.MkdirAll(filepath.Dir(entry), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(entry, []byte("[Desktop Entry]\nExec=other %u\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := l.ForwardOther("nxm://stardewvalley/mods/1", "other.desktop"); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(launched)
	if err != nil || string(b) != "launch\n"+entry+"\nnxm://stardewvalley/mods/1\n" {
		t.Fatalf("host gio got %q, %v", b, err)
	}
	if err := l.ForwardOther("nxm://x", "missing.desktop"); err == nil {
		t.Fatal("a missing entry forwarded")
	}
}

func TestFlatpakWritesHostManifestsThatStartMortarThroughFlatpak(t *testing.T) {
	l, _ := flatpakLinux(t, "")
	vivaldi := filepath.Join(hostConfig(l), "vivaldi")
	for _, d := range []string{vivaldi, filepath.Join(l.home, ".mozilla")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(l.home, ".var", "app", "tech.rethunk.Mortar", "native-host")
	if b, err := fsx.ReadFile(wrapper); err != nil || !strings.Contains(string(b), `exec flatpak run --command=mortar tech.rethunk.Mortar "$@"`) {
		t.Fatalf("wrapper %q, %v", b, err)
	}
	chromium := filepath.Join(vivaldi, "NativeMessagingHosts", nativehost.Name+".json")
	firefox := filepath.Join(l.home, ".mozilla", "native-messaging-hosts", nativehost.Name+".json")
	for _, p := range []string{chromium, firefox} {
		var m struct {
			Path string `json:"path"`
		}
		if b, err := fsx.ReadFile(p); err != nil || json.Unmarshal(b, &m) != nil || m.Path != wrapper {
			t.Fatalf("%s: %s, %v", p, b, err)
		}
	}
	for _, s := range l.NativeHostStatus() {
		if s.State != HostOK {
			t.Errorf("status %+v", s)
		}
	}
	if _, err := os.Stat(filepath.Join(l.configHome, "vivaldi")); !os.IsNotExist(err) {
		t.Errorf("wrote into the sandbox's config: %v", err)
	}
	if err := l.Restore(nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{chromium, firefox} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived Restore: %v", p, err)
		}
	}
}
