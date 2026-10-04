//go:build !windows

package shortcut

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

func TestCreateInFlatpakWritesHostEntryRunningFlatpak(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := sandbox.Getenv
	sandbox.Getenv = func(k string) string {
		if k == "FLATPAK_ID" {
			return sandbox.AppID
		}
		return ""
	}
	t.Cleanup(func() { sandbox.Getenv = old })
	path, err := create("/app/bin/mortar", Arg("stardew", "p1"), "Main")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "share", "applications"); filepath.Dir(path) != want {
		t.Fatalf("path %s, want under %s", path, want)
	}
	b, _ := fsx.ReadFile(path)
	if !strings.Contains(string(b), "Exec=flatpak run tech.rethunk.Mortar --play=stardew/p1\n") {
		t.Fatalf("entry = %s", b)
	}
	if _, err := (&Service{}).AddToSteam("stardew", "Stardew", "p1", "Main"); !errors.Is(err, ErrFlatpakSteamShortcut) {
		t.Fatalf("AddToSteam err = %v", err)
	}
}
