package sandbox

import (
	"errors"
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
		if !reflect.DeepEqual(args, []string{"--host", "which", "steam"}) {
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
