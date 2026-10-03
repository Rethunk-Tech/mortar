package nxm

import (
	"reflect"
	"testing"
)

func TestWindowsForwardArgv(t *testing.T) {
	prev := previousID(`"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe,0`, "URL:Vortex")
	link := "nxm://skyrim/mods/1/files/2"
	name, args, err := WindowsForwardArgv(prev, link)
	if err != nil {
		t.Fatal(err)
	}
	if name != `C:\Vortex\Vortex.exe` || !reflect.DeepEqual(args, []string{link}) {
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
	link := "nxm://skyrim/mods/1/files/2"
	name, args, err := LinuxForwardArgv("vortex.desktop", link, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if name != "/usr/bin/flatpak" || !reflect.DeepEqual(args, []string{"run", "com.nexusmods.vortex", link}) {
		t.Fatalf("argv = %q %v", name, args)
	}
}

func TestForwardArgvKeepsMetacharactersInOneElement(t *testing.T) {
	link := `nxm://othergame/mods/1?x=$(touch /tmp/pwned);id&` + "`uname`" + ` "quoted" spaced`
	t.Run("windows", func(t *testing.T) {
		prev := previousID(`"C:\Vortex\Vortex.exe" "%1"`, "", "")
		name, args, err := WindowsForwardArgv(prev, link)
		if err != nil {
			t.Fatal(err)
		}
		if name != `C:\Vortex\Vortex.exe` || len(args) != 1 || args[0] != link {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
	t.Run("linux", func(t *testing.T) {
		lookup := func(string) (string, error) { return `/usr/bin/vortex %u`, nil }
		name, args, err := LinuxForwardArgv("vortex.desktop", link, lookup)
		if err != nil {
			t.Fatal(err)
		}
		if name != "/usr/bin/vortex" || len(args) != 1 || args[0] != link {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
}

func TestForwardArgvQuotedExecutableWithSpaces(t *testing.T) {
	link := "nxm://skyrim/mods/1"
	t.Run("windows", func(t *testing.T) {
		prev := previousID(`"C:\Program Files\Nexus Mods\Vortex.exe" "%1"`, "", "")
		name, args, err := WindowsForwardArgv(prev, link)
		if err != nil {
			t.Fatal(err)
		}
		if name != `C:\Program Files\Nexus Mods\Vortex.exe` || !reflect.DeepEqual(args, []string{link}) {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
	t.Run("linux", func(t *testing.T) {
		lookup := func(string) (string, error) {
			return `"/usr/lib/nexus mods/vortex" --url %u`, nil
		}
		name, args, err := LinuxForwardArgv("vortex.desktop", link, lookup)
		if err != nil {
			t.Fatal(err)
		}
		if name != "/usr/lib/nexus mods/vortex" || !reflect.DeepEqual(args, []string{"--url", link}) {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
}

func TestForwardArgvAppendsWhenFieldCodeMissing(t *testing.T) {
	link := "nxm://skyrim/mods/1"
	t.Run("windows", func(t *testing.T) {
		prev := previousID(`"C:\Vortex\Vortex.exe" --nxm`, "", "")
		name, args, err := WindowsForwardArgv(prev, link)
		if err != nil {
			t.Fatal(err)
		}
		if name != `C:\Vortex\Vortex.exe` || !reflect.DeepEqual(args, []string{"--nxm", link}) {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
	t.Run("linux", func(t *testing.T) {
		lookup := func(string) (string, error) { return `/usr/bin/vortex --nxm`, nil }
		name, args, err := LinuxForwardArgv("vortex.desktop", link, lookup)
		if err != nil {
			t.Fatal(err)
		}
		if name != "/usr/bin/vortex" || !reflect.DeepEqual(args, []string{"--nxm", link}) {
			t.Fatalf("argv = %q %v", name, args)
		}
	})
}

func TestForwardArgvEmptyCommand(t *testing.T) {
	if _, _, err := WindowsForwardArgv("", "nxm://x/mods/1"); err == nil {
		t.Fatal("expected error")
	}
	lookup := func(string) (string, error) { return "  ", nil }
	if _, _, err := LinuxForwardArgv("x.desktop", "nxm://x/mods/1", lookup); err == nil {
		t.Fatal("expected error")
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
