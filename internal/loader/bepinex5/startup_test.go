package bepinex5

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestRequestStartupNamesTheTitleSceneAndReadySignal(t *testing.T) {
	dir := t.TempDir()
	if err := RequestStartup(dir, "riskofrain2", true); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(dir, measureFile))
	if err != nil || string(b) != "title\nBaseMainMenuScreen: OnEnter()" {
		t.Fatalf("request %q err %v", b, err)
	}
}
