package datadir

import (
	"errors"
	"runtime"
	"testing"
)

func TestTestsCannotReachRealData(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows resolves through LOCALAPPDATA")
	}
	t.Setenv("XDG_DATA_HOME", "/home/someone/.local/share")
	if _, err := Dir(); !errors.Is(err, errRealDataInTest) {
		t.Fatalf("a non-temporary data folder must be refused in tests, got %v", err)
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if _, err := Dir(); err != nil {
		t.Fatalf("a temporary data folder must be allowed: %v", err)
	}
}
