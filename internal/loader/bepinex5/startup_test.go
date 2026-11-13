package bepinex5

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestRequestStartupNamesTheTitleSceneAndReadySignal(t *testing.T) {
	dir := t.TempDir()
	if err := RequestStartup(dir, "riskofrain2", true); err != nil {
		t.Fatal(err)
	}
	b, err := safefs.ReadFile(filepath.Join(dir, measureFile))
	if err != nil || string(b) != "title\nBaseMainMenuScreen: OnEnter()" {
		t.Fatalf("request %q err %v", b, err)
	}
}

// Play rewrites the player's BepInEx.cfg by replacing it whole, so a crash mid-write cannot leave it cut short: a
// hardlink to the old file still reading the old text shows the switch never wrote into it.
func TestApplyConsoleReplacesBepInExCfgRatherThanWritingIntoIt(t *testing.T) {
	dir := t.TempDir()
	path := configFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	old := "[Logging.Console]\nEnabled = false\n"
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	link := path + ".before"
	if err := os.Link(path, link); err != nil {
		t.Skip("no hardlinks here:", err)
	}
	if err := applyConsole(dir, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(link); string(b) != old {
		t.Fatalf("the switch wrote into the original: %q", b)
	}
}
