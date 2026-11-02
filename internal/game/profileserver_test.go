package game

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestProfileServerStart(t *testing.T) {
	spec := components.ProfileServerSpec{Exe: "S.exe", Assembly: "S.dll", DataFlag: "--dataPath"}
	exe, args, err := ProfileServerStart(spec, "windows", "/g", "/d")
	if err != nil || exe != filepath.Join("/g", "S.exe") || !slices.Equal(args, []string{"--dataPath", "/d"}) {
		t.Fatalf("windows: %q %v %v", exe, args, err)
	}
	if runtime.GOOS == "windows" {
		return // the linux path finds an extensionless dotnet on PATH, which a Windows host never resolves
	}
	bin := t.TempDir()
	testfs.WriteAt(t, filepath.Join(bin, "dotnet"), []byte("#!/bin/sh\n"))
	if err := safefs.Chmod(filepath.Join(bin, "dotnet"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	exe, args, err = ProfileServerStart(spec, "linux", "/g", "/d")
	if err != nil || exe != filepath.Join(bin, "dotnet") || !slices.Equal(args, []string{filepath.Join("/g", "S.dll"), "--dataPath", "/d"}) {
		t.Fatalf("linux: %q %v %v", exe, args, err)
	}
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if _, _, err := ProfileServerStart(spec, "linux", "/g", "/d"); err == nil {
		t.Fatal("no dotnet, no error")
	}
}
