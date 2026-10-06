package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckOpenableRefusesWhatIsNotALocalFolderOrFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{dir: true, file: true, "": false, "relative/dir": false, "-rf": false, filepath.Join(dir, "missing"): false}
	if runtime.GOOS == "windows" {
		cases[`\\server\share\tool.exe`] = false
	}
	if runtime.GOOS != "windows" {
		toFile, toDir := filepath.Join(dir, "link-file"), filepath.Join(dir, "link-dir")
		if err := os.Symlink("/etc/hostname", toFile); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dir, toDir); err != nil {
			t.Fatal(err)
		}
		cases[toFile], cases[toDir] = false, true
	}
	for path, ok := range cases {
		if err := checkOpenable(path); (err == nil) != ok {
			t.Errorf("checkOpenable(%q) = %v, want ok=%v", path, err, ok)
		}
	}
}

func TestIsUNCNamesNetworkSharesInBothSpellings(t *testing.T) {
	for path, want := range map[string]bool{
		`\\server\share\x.exe`: true, `\\?\UNC\server\share`: true, `\\?\unc\server\share`: true,
		`\\?\C:\Users\me`: false, `C:\Users\me`: false, `/home/me`: false,
	} {
		if got := isUNC(path); got != want {
			t.Errorf("isUNC(%q) = %v", path, got)
		}
	}
}
