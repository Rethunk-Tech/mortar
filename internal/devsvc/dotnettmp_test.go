package devsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSweepDotnetTempRemovesOnlyOldPerCommandFolders(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for _, name := range []string{"aB3xYz", "Q9wErt", "MSBuildTemp", "MSBuildTempAb12Cd", "MSBuildTempOld123", "keep.txt"} {
		path := filepath.Join(dir, name)
		if name == "keep.txt" {
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.MkdirAll(filepath.Join(path, "inner"), 0o700); err != nil {
			t.Fatal(err)
		}
		when := old
		if name == "MSBuildTempOld123" {
			when = time.Now().Add(-48 * time.Hour)
		}
		if name != "Q9wErt" {
			if err := os.Chtimes(path, when, when); err != nil {
				t.Fatal(err)
			}
		}
	}
	sweepDotnetTemp(dir, time.Now().Add(-dotnetLeftoverAge), time.Now().Add(-msbuildTempAge))
	for name, want := range map[string]bool{
		"aB3xYz": false, "Q9wErt": true, "MSBuildTemp": true, "MSBuildTempAb12Cd": true, "MSBuildTempOld123": false, "keep.txt": true,
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); (err == nil) != want {
			t.Errorf("%s: present %v, want %v", name, err == nil, want)
		}
	}
}

func TestDotnetRunsWithItsTempFolderInTheCache(t *testing.T) {
	s := &Service{CacheDir: t.TempDir()}
	env := s.dotnetEnv()
	want := "TMPDIR=" + filepath.Join(s.CacheDir, "dotnet-tmp")
	found := false
	for _, kv := range env {
		found = found || kv == want
	}
	if !found {
		t.Fatalf("no %s in the environment", want)
	}
	if _, err := os.Stat(filepath.Join(s.CacheDir, "dotnet-tmp")); err != nil {
		t.Fatal(err)
	}
}

func TestDotnetEnvDropsAppImageLibraries(t *testing.T) {
	t.Setenv("APPIMAGE", "/home/u/Mortar.AppImage")
	t.Setenv("APPDIR", "/tmp/.mount_mortarX")
	t.Setenv("LD_LIBRARY_PATH", "/tmp/.mount_mortarX/usr/lib")
	for _, kv := range (&Service{}).dotnetEnv() {
		if strings.Contains(kv, "/tmp/.mount_mortarX") {
			t.Errorf("build environment keeps %s", kv)
		}
	}
}
