package smapi

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAWindowsBuildOnALinuxHostInstallsTheWindowsWay(t *testing.T) {
	bottle, native := t.TempDir(), t.TempDir()
	testfs.WriteFile(t, bottle, "Stardew Valley.exe", "")
	testfs.WriteFile(t, native, "Stardew Valley", "")
	if got := buildOS("linux", bottle); got != "windows" {
		t.Errorf("bottle build = %q", got)
	}
	if got := buildOS("linux", native); got != "linux" {
		t.Errorf("native build = %q", got)
	}
	if got := buildOS("windows", bottle); got != "windows" {
		t.Errorf("windows host = %q", got)
	}
}
