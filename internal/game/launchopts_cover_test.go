package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
)

func TestSetAndClearLaunchOptionUnknownGame(t *testing.T) {
	s := &Service{}
	if _, err := s.SetLaunchOption("not-a-game"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := s.ClearLaunchOption("not-a-game"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSetLaunchOptionNotInstalled(t *testing.T) {
	s := NewService(t.TempDir(), testStore(t))
	_, err := s.SetLaunchOption("stardew")
	if err == nil || !strings.Contains(err.Error(), "is not installed") {
		t.Fatalf("got %v", err)
	}
}

func TestClearLaunchOptionAndClosedSteamWithoutSteam(t *testing.T) {
	s := NewService(t.TempDir(), testStore(t))
	if _, err := s.closedSteam(); err == nil || !strings.Contains(err.Error(), "steam was not found") {
		t.Fatalf("closedSteam: %v", err)
	}
	if _, err := s.ClearLaunchOption("stardew"); err == nil || !strings.Contains(err.Error(), "steam was not found") {
		t.Fatalf("ClearLaunchOption: %v", err)
	}
}

func TestSetAndClearLaunchOptionWithSteamHome(t *testing.T) {
	h := home(t)
	s := NewService(h, steamStore(t, h))
	if _, err := s.closedSteam(); err != nil && strings.Contains(err.Error(), "close Steam first") {
		// Steam is running: both calls refuse rather than edit a config Steam would rewrite on exit.
		if _, err := s.SetLaunchOption("stardew"); err == nil || !strings.Contains(err.Error(), "close Steam first") {
			t.Fatalf("SetLaunchOption with Steam running: %v", err)
		}
		if _, err := s.ClearLaunchOption("stardew"); err == nil || !strings.Contains(err.Error(), "close Steam first") {
			t.Fatalf("ClearLaunchOption with Steam running: %v", err)
		}
		return
	}
	st, status := LocateSteam(h, s.store.Get())
	if status != steam.Found {
		// This OS looks for Steam somewhere a temporary home does not hold it (macOS): both calls report Steam
		// missing instead of editing anything.
		if _, err := s.SetLaunchOption("stardew"); err == nil || !strings.Contains(err.Error(), "steam was not found") {
			t.Fatalf("SetLaunchOption without Steam: %v", err)
		}
		if _, err := s.ClearLaunchOption("stardew"); err == nil || !strings.Contains(err.Error(), "steam was not found") {
			t.Fatalf("ClearLaunchOption without Steam: %v", err)
		}
		return
	}
	// Steam keeps launch options in the signed-in account's userdata config folder, named after the account's
	// SteamID64 less Steam's base id. The files go under the root LocateSteam found, the path the code itself
	// reads, so the fixture cannot drift from the folder the code looks in.
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const account, userdataDir = "76561197960265729", "1"
	write(filepath.Join(st.Root, "config", "loginusers.vdf"),
		`"users" { "`+account+`" { "AccountName" "a" "MostRecent" "1" } }`)
	write(filepath.Join(st.Root, "userdata", userdataDir, "config", "localconfig.vdf"),
		`"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "" } } } } } }`)

	set, err := s.SetLaunchOption("stardew")
	if err != nil || !strings.Contains(set, "%command%") {
		t.Fatalf("SetLaunchOption = %q, %v", set, err)
	}
	if got, err := s.LaunchOptions("stardew"); err != nil || got != set {
		t.Fatalf("LaunchOptions = %q, %v; want %q", got, err, set)
	}
	cleared, err := s.ClearLaunchOption("stardew")
	if err != nil || cleared != "" {
		t.Fatalf("ClearLaunchOption = %q, %v", cleared, err)
	}
	if got, err := s.LaunchOptions("stardew"); err != nil || got != "" {
		t.Fatalf("LaunchOptions after clear = %q, %v", got, err)
	}
}

func TestAnnotateLaunchOptionsErr(t *testing.T) {
	out, err := annotateLaunchOptionsErr("kept", steam.ErrNoAppsSection)
	if out != "kept" {
		t.Fatalf("out=%q", out)
	}
	if !errors.Is(err, steam.ErrNoAppsSection) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "launch the game from Steam once first") {
		t.Fatalf("got %v", err)
	}

	other := errors.New("disk")
	out, err = annotateLaunchOptionsErr("y", other)
	if out != "y" || !errors.Is(err, other) {
		t.Fatalf("out=%q err=%v", out, err)
	}

	out, err = annotateLaunchOptionsErr("", nil)
	if out != "" || err != nil {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
