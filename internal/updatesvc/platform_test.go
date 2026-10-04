package updatesvc

import "testing"

func TestUpdatePlatform(t *testing.T) {
	for _, c := range []struct {
		goos, exe, launchable, want string
	}{
		{"linux", "/home/u/mortar-linux-amd64", "/home/u/mortar-linux-amd64", PortablePlatform},
		{"linux", "/tmp/.mount_mortarX/usr/bin/mortar", "/home/u/Mortar.AppImage", ""},
		{"windows", `C:\Mortar\mortar.exe`, `C:\Mortar\mortar.exe`, ""},
	} {
		if got := updatePlatform(c.goos, c.exe, c.launchable); got != c.want {
			t.Errorf("updatePlatform(%q, %q, %q) = %q, want %q", c.goos, c.exe, c.launchable, got, c.want)
		}
	}
}
