package tools

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLaunchRefusesAWindowsProgramAndStartsALinuxOne(t *testing.T) {
	gameDir := t.TempDir()
	if runtime.GOOS == "linux" {
		if err := launchTool(Tool{Name: "w", Executable: filepath.Join(gameDir, "t.exe")}, Context{Game: gameDir}); err == nil {
			t.Fatal(".exe accepted on Linux")
		}
	}
	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true in PATH")
	}
	if err := launchTool(Tool{Name: "true", Executable: truePath}, Context{Game: gameDir}); err != nil {
		t.Fatalf("linux program: %v", err)
	}
}

func TestEmptyPathStaysEmptyAndARelativeOneBecomesAbsolute(t *testing.T) {
	if absDir("") != "" {
		t.Fatal("empty")
	}
	got := absDir(".")
	if !filepath.IsAbs(got) {
		t.Fatalf("relative = %q", got)
	}
}
