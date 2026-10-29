package sandbox

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHostArgv(t *testing.T) {
	name, args := HostArgv("/g", []string{"A=1"}, "steam", "-applaunch", "413150")
	want := []string{"--host", "--directory=/g", "--env=A=1", "steam", "-applaunch", "413150"}
	if name != "flatpak-spawn" || !reflect.DeepEqual(args, want) {
		t.Fatalf("%s %v", name, args)
	}
	_, args = HostArgv("", nil, "which", "steam")
	if !reflect.DeepEqual(args, []string{"--host", "which", "steam"}) {
		t.Fatalf("%v", args)
	}
}

func TestHostLookPath(t *testing.T) {
	t.Cleanup(func(old func(...string) ([]byte, error)) func() { return func() { Output = old } }(Output))
	Output = func(args ...string) ([]byte, error) {
		// The host's sh, not which: minimal hosts (Arch's base) have no which.
		if !reflect.DeepEqual(args, []string{"--host", "sh", "-c", `command -v -- "$1"`, "sh", "steam"}) {
			t.Fatalf("%v", args)
		}
		return []byte("/usr/bin/steam\n"), nil
	}
	if p, err := HostLookPath("steam"); err != nil || p != "/usr/bin/steam" {
		t.Fatalf("%q %v", p, err)
	}
	Output = func(...string) ([]byte, error) { return nil, errors.New("exit 1") }
	if _, err := HostLookPath("steam"); err == nil {
		t.Fatal("want error")
	}
}

func TestInFlatpak(t *testing.T) {
	t.Cleanup(func(old func(string) string) func() { return func() { Getenv = old } }(Getenv))
	Getenv = func(string) string { return "" }
	if InFlatpak() {
		t.Fatal("no FLATPAK_ID")
	}
	Getenv = func(k string) string { return map[string]string{"FLATPAK_ID": AppID}[k] }
	if !InFlatpak() {
		t.Fatal("FLATPAK_ID set")
	}
}

func TestHostXDGHomes(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(t.TempDir(), "cfg")
	t.Setenv("FLATPAK_ID", AppID)
	t.Setenv("HOST_XDG_CONFIG_HOME", custom)
	t.Setenv("HOST_XDG_DATA_HOME", "relative/data")
	if got := HostConfigHome(home); got != custom {
		t.Fatalf("config %q", got)
	}
	if got, want := HostDataHome(home), filepath.Join(home, ".local", "share"); got != want {
		t.Fatalf("relative data %q, want %q", got, want)
	}
	t.Setenv("FLATPAK_ID", "")
	if got, want := HostConfigHome(home), filepath.Join(home, ".config"); got != want {
		t.Fatalf("outside flatpak %q, want %q", got, want)
	}
}
