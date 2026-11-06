package dotnetroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestRoot(t *testing.T) {
	mount := t.TempDir()
	if got := Root(true, mount); got != "" {
		t.Fatalf("empty mount resolved to %q", got)
	}
	lib := filepath.Join(mount, "lib")
	if err := os.MkdirAll(lib, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "dotnet"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Root(true, mount); got != lib {
		t.Fatalf("got %q, want %q", got, lib)
	}
	if got := Root(false, mount); got != "" {
		t.Fatalf("outside a Flatpak resolved to %q", got)
	}
}

func TestWithRoot(t *testing.T) {
	env := []string{"A=1", "PATH=/usr/bin", "DOTNET_ROOT=/old"}
	got := WithRoot(env, "/ext/lib")
	for _, want := range []string{"A=1", "DOTNET_ROOT=/ext/lib", "PATH=/ext/lib" + string(os.PathListSeparator) + "/usr/bin"} {
		if !slices.Contains(got, want) {
			t.Errorf("%q missing from %v", want, got)
		}
	}
	if slices.Contains(got, "DOTNET_ROOT=/old") {
		t.Errorf("old DOTNET_ROOT kept: %v", got)
	}
	if !slices.Equal(WithRoot(env, ""), env) {
		t.Error("no root must leave env alone")
	}
	if Program("") != "dotnet" || Program("/ext/lib") != filepath.Join("/ext/lib", "dotnet") {
		t.Error("Program")
	}
}

func TestHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	want := filepath.Join(home, ".dotnet", "dotnet")
	var asked []string
	look := func(found bool) func(string) (string, error) {
		return func(p string) (string, error) {
			asked = append(asked, p)
			if found {
				return p, nil
			}
			return "", exec.ErrNotFound
		}
	}
	if got := Home(look(false)); got != "" {
		t.Fatalf("missing dotnet resolved to %q", got)
	}
	if got := Home(look(true)); got != filepath.Join(home, ".dotnet") {
		t.Fatalf("got %q", got)
	}
	if len(asked) != 2 || asked[0] != want {
		t.Fatalf("lookPath asked %q, want %q", asked, want)
	}
}
