package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchOptions(t *testing.T) {
	h := home(t)
	svc := NewService(h, testStore(t))
	if _, err := svc.LaunchOptions("nope"); err == nil {
		t.Fatal("unknown game should fail")
	}
	if _, err := svc.LaunchOptions("stardew"); err == nil {
		t.Fatal("unreadable config should fail")
	}
	if got, err := NewService(t.TempDir(), testStore(t)).LaunchOptions("stardew"); err != nil || got != "" {
		t.Fatalf("no steam = %q, %v", got, err)
	}

	root := filepath.Join(h, ".local", "share", "Steam")
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("config/loginusers.vdf", `"users" { "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`)
	write("userdata/39734274/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" {
		"413150" { "LaunchOptions" "\"C:\\Stardew Valley\\StardewModdingAPI.exe\" %command%" } } } } } }`)
	got, err := svc.LaunchOptions("stardew")
	if err != nil || got != `"C:\Stardew Valley\StardewModdingAPI.exe" %command%` {
		t.Fatalf("launch options = %q, %v", got, err)
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
