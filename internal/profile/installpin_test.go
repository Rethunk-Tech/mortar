package profile

import "testing"

func TestProfileInstallPinRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
