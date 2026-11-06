package devsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExtensionDotnetRoot(t *testing.T) {
	mount := t.TempDir()
	if got := extensionDotnetRoot(true, mount); got != "" {
		t.Fatalf("empty mount resolved to %q", got)
	}
	lib := filepath.Join(mount, "lib")
	if err := os.MkdirAll(lib, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "dotnet"), nil, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := extensionDotnetRoot(true, mount); got != lib {
		t.Fatalf("got %q, want %q", got, lib)
	}
	if got := extensionDotnetRoot(false, mount); got != "" {
		t.Fatalf("outside a Flatpak resolved to %q", got)
	}
}

func TestWithDotnetRoot(t *testing.T) {
	env := []string{"A=1", "PATH=/usr/bin", "DOTNET_ROOT=/old"}
	got := withDotnetRoot(env, "/ext/lib")
	for _, want := range []string{"A=1", "DOTNET_ROOT=/ext/lib", "PATH=/ext/lib" + string(os.PathListSeparator) + "/usr/bin"} {
		if !slices.Contains(got, want) {
			t.Errorf("%q missing from %v", want, got)
		}
	}
	if slices.Contains(got, "DOTNET_ROOT=/old") {
		t.Errorf("old DOTNET_ROOT kept: %v", got)
	}
	if !slices.Equal(withDotnetRoot(env, ""), env) {
		t.Error("no root must leave env alone")
	}
	if dotnetProgram("") != "dotnet" || dotnetProgram("/ext/lib") != filepath.Join("/ext/lib", "dotnet") {
		t.Error("dotnetProgram")
	}
}
