package nxm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// flatpakLinux is newLinux inside a Flatpak whose host is this machine: a host command runs here, against the test's
// folders, and a host gio only records what it was asked to launch.
func flatpakLinux(t *testing.T, current string) (*System, *recorder, string) {
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
	t.Setenv("XDG_CONFIG_HOME", l.configHome)
	t.Setenv("XDG_DATA_HOME", l.dataHome)
	t.Setenv("XDG_DATA_DIRS", t.TempDir())
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	return l, r, launched
}

func TestFlatpakForwardsToThePreviousHandlerOnTheHost(t *testing.T) {
	l, _, launched := flatpakLinux(t, "")
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
