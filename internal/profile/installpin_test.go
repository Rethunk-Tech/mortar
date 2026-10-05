package profile

import "testing"

func TestProfileInstallPinRoundTrips(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if got := e.InstallOf("stardew", p.ID); got != "" {
		t.Fatalf("new profile pinned to %q", got)
	}
	if _, err := e.SetInstall("stardew", p.ID, "abc"); err != nil {
		t.Fatal(err)
	}
	if got := e.InstallOf("stardew", p.ID); got != "abc" {
		t.Fatalf("pin = %q", got)
	}
}

func TestSeparateSavesFlagRoundTrips(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if e.SeparateSaves("stardew", p.ID) {
		t.Fatal("a new profile shares the game's saves")
	}
	if _, err := e.SetSeparateSaves("stardew", p.ID, true, false); err != nil {
		t.Fatal(err)
	}
	if !e.SeparateSaves("stardew", p.ID) {
		t.Fatal("the flag must persist")
	}
}
