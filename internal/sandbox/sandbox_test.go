package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
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
	Getenv = func(k string) string { return map[string]string{"FLATPAK_ID": appversion.AppID}[k] }
	if !InFlatpak() {
		t.Fatal("FLATPAK_ID set")
	}
}

func TestHostXDGHomes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := filepath.Join(t.TempDir(), "cfg")
	t.Setenv("FLATPAK_ID", appversion.AppID)
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
	t.Setenv("XDG_CONFIG_HOME", custom)
	t.Setenv("XDG_DATA_HOME", "relative/data")
	if got := HostConfigHome(home); got != custom {
		t.Fatalf("custom XDG_CONFIG_HOME %q, want %q", got, custom)
	}
	if got, want := HostDataHome(home), filepath.Join(home, ".local", "share"); got != want {
		t.Fatalf("relative XDG_DATA_HOME %q, want %q", got, want)
	}
}

func TestCommandContextHostTerm(t *testing.T) {
	name, args := HostArgv("", nil, "sleep", "1")
	cmd := CommandContext(context.Background(), name, args...)
	if cmd.Cancel == nil || cmd.WaitDelay == 0 {
		t.Fatalf("host command lacks Cancel/WaitDelay: %v %v", cmd.Cancel == nil, cmd.WaitDelay)
	}
	if plain := CommandContext(context.Background(), "sleep", "1"); plain.WaitDelay != 0 {
		t.Fatal("non-host command should keep the default kill")
	}
}

func TestHostLookPathRejectsBuiltin(t *testing.T) {
	old := Output
	t.Cleanup(func() { Output = old })
	Output = func(...string) ([]byte, error) { return []byte("cd\n"), nil }
	if p, err := HostLookPath("cd"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("HostLookPath(cd) = %q, %v; want ErrNotFound", p, err)
	}
	Output = func(...string) ([]byte, error) { return []byte("/usr/bin/ls\n"), nil }
	if p, err := HostLookPath("ls"); err != nil || p != "/usr/bin/ls" {
		t.Fatalf("HostLookPath(ls) = %q, %v", p, err)
	}
}

func TestHostHomesResolveEmptyHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	t.Chdir(t.TempDir())
	if got, want := HostConfigHome(""), filepath.Join(home, ".config"); got != want {
		t.Errorf("HostConfigHome(\"\") = %q, want %q", got, want)
	}
	if got, want := HostDataHome(""), filepath.Join(home, ".local", "share"); got != want {
		t.Errorf("HostDataHome(\"\") = %q, want %q", got, want)
	}
}
