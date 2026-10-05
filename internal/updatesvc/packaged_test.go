package updatesvc

import "testing"

func TestPackagedBy(t *testing.T) {
	for exe, want := range map[string]string{
		`C:\Users\a\scoop\apps\mortar\0.2.1\mortar.exe`:             "scoop",
		`D:\Scoop\apps\mortar\current\mortar.exe`:                   "scoop",
		"/home/linuxbrew/.linuxbrew/Cellar/mortar/0.2.1/bin/mortar": "homebrew",
		"/home/a/.linuxbrew/bin/mortar":                             "homebrew",
		`C:\Users\a\AppData\Local\Programs\Mortar\mortar.exe`:       "",
		"/home/a/Applications/mortar-linux-x86_64.AppImage":         "",
	} {
		if got := PackagedBy(exe); got != want {
			t.Errorf("PackagedBy(%q) = %q, want %q", exe, got, want)
		}
	}
}
