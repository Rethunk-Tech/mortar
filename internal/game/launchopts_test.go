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
