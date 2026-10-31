package game

import (
	"testing"
)

func TestLaunchOptions(t *testing.T) {
	h := home(t)
	svc := NewService(h, steamStore(t, h))
	if _, err := svc.LaunchOptions("nope"); err == nil {
		t.Fatal("unknown game should fail")
	}
	if _, err := svc.LaunchOptions("stardew"); err == nil {
		t.Fatal("unreadable config should fail")
	}
	if got, err := NewService(t.TempDir(), testStore(t)).LaunchOptions("stardew"); err != nil || got != "" {
		t.Fatalf("no steam = %q, %v", got, err)
	}
}

func TestOnlyALoaderSteamStartsNeedsLaunchOptions(t *testing.T) {
	for _, tc := range []struct {
		game  string
		needs bool
	}{{"stardew", true}, {"lethal-company", false}, {"valheim", false}} {
		needs, err := NeedsLaunchOption(tc.game)
		if err != nil || needs != tc.needs {
			t.Fatalf("%s needs = %v, %v", tc.game, needs, err)
		}
		// A vanilla Play asks whether the options force the loader; BepInEx games must get a plain no, not an error.
		starts, err := StartsLoader(tc.game, `"C:\Games\StardewModdingAPI.exe" %command%`)
		if err != nil || starts != tc.needs {
			t.Fatalf("%s starts = %v, %v", tc.game, starts, err)
		}
	}
	if _, err := NeedsLaunchOption("nope"); err == nil {
		t.Fatal("unknown game should fail")
	}
}
