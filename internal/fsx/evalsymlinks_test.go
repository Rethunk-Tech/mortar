package fsx

import (
	"runtime"
	"strings"
	"testing"
)

func TestExtendedPathOnlyPrefixesLongDrivePaths(t *testing.T) {
	long := `C:\Users\u\AppData\Local\Mortar\store\blobs\` + strings.Repeat("a", 64) + `\` + strings.Repeat(`d\`, 80) + "f.dll"
	for _, tc := range []struct{ in, want string }{
		{`C:\short\path`, `C:\short\path`},
		{`\\server\share\` + strings.Repeat("x", 300), `\\server\share\` + strings.Repeat("x", 300)},
		{`\\?\C:\` + strings.Repeat("x", 300), `\\?\C:\` + strings.Repeat("x", 300)},
		{long, `\\?\` + long},
	} {
		if got := ExtendedPath(tc.in); got != tc.want {
			t.Errorf("ExtendedPath(%.40q) = %.40q, want %.40q", tc.in, got, tc.want)
		}
	}
}

func TestSamePath(t *testing.T) {
	if !SamePath(`/games/Stardew Valley/`, `/games/Stardew Valley`) {
		t.Fatal("a trailing separator made a different path")
	}
	differentCase := SamePath(`D:\SteamLibrary\steamapps\common\Stardew Valley`, `d:\steamlibrary\steamapps\common\stardew valley`)
	if differentCase != (runtime.GOOS == "windows") {
		t.Fatalf("case-only difference: same = %v on %s", differentCase, runtime.GOOS)
	}
}
