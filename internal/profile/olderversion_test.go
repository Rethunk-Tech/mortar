package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
)

func TestDownloadDepotCommand(t *testing.T) {
	got, err := downloadDepotCommand("413150", "413151", "7081337227611034738")
	if err != nil || got != "download_depot 413150 413151 7081337227611034738" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range [][3]string{{"413150", "413151", "1; quit"}, {"x", "1", "2"}, {"1", "", "2"}} {
		if _, err := downloadDepotCommand(bad[0], bad[1], bad[2]); err == nil {
			t.Fatalf("%v accepted", bad)
		}
	}
}

func TestDepotPaths(t *testing.T) {
	lib := filepath.Join("home", "u", "SteamLibrary")
	if got := libraryOf(filepath.Join(lib, "steamapps", "common", "Stardew Valley")); got != lib {
		t.Fatalf("libraryOf = %q", got)
	}
	if got := libraryOf(filepath.Join("home", "u", "Games", "Stardew")); got != "" {
		t.Fatalf("a folder outside a library has library %q", got)
	}
	if got, want := steam.DepotDownloadDir(lib, "413150", "413151"), filepath.Join(lib, "steamapps", "content", "app_413150", "depot_413151"); got != want {
		t.Fatalf("DepotDownloadDir = %q, want %q", got, want)
	}
	if got, want := versionFolder("/data", "stardew", "99-20260101"), filepath.Join("/data", "versions", "stardew", "99-20260101"); got != want {
		t.Fatalf("versionFolder = %q", got)
	}
}
