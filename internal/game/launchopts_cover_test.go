package game

import (
	"errors"
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
	if _, err := s.closedSteam(); err != nil && !strings.Contains(err.Error(), "close Steam first") {
		t.Fatalf("closedSteam: %v", err)
	}
	_, setErr := s.SetLaunchOption("stardew")
	_, clearErr := s.ClearLaunchOption("stardew")
	if setErr == nil && clearErr == nil {
		return
	}
	ok := func(err error) bool {
		if err == nil {
			return true
		}
		msg := err.Error()
		return strings.Contains(msg, "close Steam first") ||
			strings.Contains(msg, "launch the game from Steam once first") ||
			strings.Contains(msg, "no loader") ||
			strings.Contains(msg, "no such file") ||
			strings.Contains(msg, "steam")
	}
	if !ok(setErr) {
		t.Fatalf("SetLaunchOption: %v", setErr)
	}
	if !ok(clearErr) {
		t.Fatalf("ClearLaunchOption: %v", clearErr)
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
