package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestSteamArgsCarryTheSessionFlag(t *testing.T) {
	if got := SteamArgs("stardew", "p1"); got != "--play=stardew/p1 --steam-session" {
		t.Fatalf("SteamArgs = %q", got)
	}
	if !SteamSession([]string{"--other", SteamSessionFlag}) {
		t.Fatal("SteamSession must find the flag among args")
	}
	if SteamSession([]string{Arg("stardew", "p1")}) {
		t.Fatal("a play argument alone is not a Steam session")
	}
}

func TestCapabilitiesHideAddToSteamInsideAFlatpak(t *testing.T) {
	got := (&Service{}).Capabilities()
	if !got.Shortcuts {
		t.Fatal("desktop shortcuts stay available")
	}
	if got.AddToSteam == got.Flatpak {
		t.Fatalf("AddToSteam must be the opposite of Flatpak, got %+v", got)
	}
}

func TestCreateRejectsAMissingGameOrProfile(t *testing.T) {
	s := &Service{}
	for _, in := range [][2]string{{"", "p"}, {"g", ""}, {"g/x", "p"}, {"g", `p"q`}} {
		if _, err := s.Create(in[0], "Game", in[1], "Profile"); err == nil {
			t.Errorf("Create(%q, %q) must be refused", in[0], in[1])
		}
	}
}

func TestCreateWritesAShortcutNamedAfterTheProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows shortcut goes through the shell")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path, err := (&Service{}).Create("stardew", "Stardew Valley", "p1", "Main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestFirstLocalFileSkipsEmptyURLsAndMissingPaths(t *testing.T) {
	if got := firstLocalFile(nil); got != "" {
		t.Fatalf("empty list = %q", got)
	}
	if got := firstLocalFile([]string{"", "https://example/cover.png", filepath.Join(t.TempDir(), "gone.png")}); got != "" {
		t.Fatalf("nothing usable = %q", got)
	}
	want := filepath.Join(t.TempDir(), "cover.png")
	if err := safefs.WriteFile(want, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := firstLocalFile([]string{"https://example/cover.png", want}); got != want {
		t.Fatalf("first local file = %q, want %q", got, want)
	}
}

func TestAddToSteamRejectsAMissingGameOrProfile(t *testing.T) {
	s := &Service{}
	if _, err := s.AddToSteam("", "Game", "p", "Profile"); err == nil {
		t.Fatal("empty game must be refused")
	}
	if _, err := s.AddToSteam("g", "Game", "", "Profile"); err == nil {
		t.Fatal("empty profile must be refused")
	}
}

func TestAddToSteamRefusesAFlatpak(t *testing.T) {
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")
	if _, err := (&Service{}).AddToSteam("stardew", "Stardew Valley", "p1", "Main"); !errors.Is(err, ErrFlatpakSteamShortcut) {
		t.Fatalf("got %v", err)
	}
	got := (&Service{}).Capabilities()
	if !got.Flatpak || got.AddToSteam {
		t.Fatalf("inside a Flatpak: %+v", got)
	}
}
