package bepinex5

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequestStartupNamesTheTitleSceneAndReadySignal(t *testing.T) {
	dir := t.TempDir()
	if err := RequestStartup(dir, "riskofrain2", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, measureFile))
	if err != nil || string(b) != "title\nBaseMainMenuScreen: OnEnter()" {
		t.Fatalf("request %q err %v", b, err)
	}
}
