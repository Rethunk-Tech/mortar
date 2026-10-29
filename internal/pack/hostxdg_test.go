package pack

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestManagerDataDirsFollowHostXDGHomes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the host XDG homes are a Flatpak sandbox escape, which Linux alone has")
	}
	home := t.TempDir()
	host := t.TempDir()
	r2 := filepath.Join(host, "config", "r2modmanPlus-local")
	gale := filepath.Join(host, "data", galeAppID)
	for _, d := range []string{r2, gale} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".var", "app", "x", "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".var", "app", "x", "data"))
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")
	t.Setenv("HOST_XDG_CONFIG_HOME", filepath.Join(host, "config"))
	t.Setenv("HOST_XDG_DATA_HOME", filepath.Join(host, "data"))
	if got := DataDirs(""); !slices.Contains(got, r2) {
		t.Fatalf("r2modman dirs %v lack %s", got, r2)
	}
	if got := GaleDataDirs(""); !slices.Contains(got, gale) {
		t.Fatalf("gale dirs %v lack %s", got, gale)
	}
	t.Setenv("HOST_XDG_CONFIG_HOME", "relative")
	t.Setenv("HOST_XDG_DATA_HOME", "relative")
	if got := DataDirs(""); slices.Contains(got, r2) {
		t.Fatalf("relative HOST_XDG_CONFIG_HOME honoured: %v", got)
	}
	if got := GaleDataDirs(""); slices.Contains(got, gale) {
		t.Fatalf("relative HOST_XDG_DATA_HOME honoured: %v", got)
	}
}
