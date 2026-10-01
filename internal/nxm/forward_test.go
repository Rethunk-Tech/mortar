package nxm

import (
	"reflect"
	"testing"
)

func TestWindowsForwardArgv(t *testing.T) {
	prev := previousID(`"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe,0`, "URL:Vortex")
	name, args, err := WindowsForwardArgv(prev, "nxm://skyrim/mods/1/files/2")
	if err != nil {
		t.Fatal(err)
	}
	if name != "cmd" || !reflect.DeepEqual(args, []string{"/c", `"C:\Vortex\Vortex.exe" "nxm://skyrim/mods/1/files/2"`}) {
		t.Fatalf("argv = %q %v", name, args)
	}
}

func TestLinuxForwardArgv(t *testing.T) {
	lookup := func(id string) (string, error) {
		if id != "vortex.desktop" {
			t.Fatalf("id = %q", id)
		}
		return `"/usr/bin/flatpak" run com.nexusmods.vortex %u`, nil
	}
	name, args, err := LinuxForwardArgv("vortex.desktop", "nxm://skyrim/mods/1/files/2", lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := `"/usr/bin/flatpak" run com.nexusmods.vortex nxm://skyrim/mods/1/files/2`
	if name != "sh" || len(args) != 2 || args[1] != want {
		t.Fatalf("argv = %q %v", name, args)
	}
}

func TestLinkGame(t *testing.T) {
	got, err := LinkGame("nxm://skyrim/mods/1/files/2?key=k&expires=1&user_id=1")
	if err != nil || got != "skyrim" {
		t.Fatalf("game = %q, %v", got, err)
	}
	if _, err := LinkGame("https://x"); err == nil {
		t.Fatal("expected form error")
	}
}
