package smapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestArmIntroSkipWritesTheBridgesRequest(t *testing.T) {
	dir := t.TempDir()
	var l loader.IntroSkipper = Loader{}
	if err := l.ArmIntroSkip(dir, loader.IntroAnimations); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "startup", "skip-intro")
	if b, err := safefs.ReadFile(path); err != nil || string(b) != "intro" {
		t.Fatalf("request %q, %v", b, err)
	}
	if err := l.ArmIntroSkip(dir, loader.IntroPlay); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the request outlived a normal start: %v", err)
	}
}
