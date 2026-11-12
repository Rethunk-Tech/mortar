package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
)

type shareLoader struct{ Loader }

func (shareLoader) SharesLog() {}

func TestLiveOfTrustsOnlyGameVersion(t *testing.T) {
	status := func(source string) bridge.Status {
		var st bridge.Status
		body := `{"gameVersion":"1.2","gameVersionSource":"` + source + `","scene":"Menu","plugins":[{"guid":"a.b","version":"3","location":"x/y.dll"}]}`
		if err := json.Unmarshal([]byte(body), &st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	live := LiveOf(status("game"))
	if live.GameVersion != "1.2" || live.Scene != "Menu" || len(live.Plugins) != 1 || live.Plugins[0] != (LivePlugin{"a.b", "3", "x/y.dll"}) {
		t.Errorf("LiveOf = %+v", live)
	}
	if v := LiveOf(status("engine")).GameVersion; v != "" {
		t.Errorf("engine default version kept: %q", v)
	}
}

func TestDebuggerAndSharesLog(t *testing.T) {
	if Debugger(t.Context()) || !Debugger(WithDebugger(t.Context())) {
		t.Error("Debugger flag does not follow WithDebugger")
	}
	if SharesLog(fakeLoader{}) || !SharesLog(shareLoader{}) {
		t.Error("SharesLog does not follow the LogShare interface")
	}
}

func TestWriteIntroSkipWritesAndClears(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "startup", "skip-intro")
	if err := WriteIntroSkip(dir, IntroToMenu); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Clean(path)); err != nil || string(b) != "menu" {
		t.Fatalf("file = %q, %v", b, err)
	}
	if err := WriteIntroSkip(dir, IntroPlay); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("IntroPlay left the file: %v", err)
	}
	if err := WriteIntroSkip(dir, IntroPlay); err != nil {
		t.Errorf("clearing an absent request failed: %v", err)
	}
}

func TestAllListsLoadersByID(t *testing.T) {
	Register(fakeLoader{id: "all-b"})
	Register(fakeLoader{id: "all-a"})
	var seen []string
	for _, l := range All() {
		if id := l.ID(); id == "all-a" || id == "all-b" {
			seen = append(seen, id)
		}
	}
	if len(seen) != 2 || seen[0] != "all-a" {
		t.Errorf("All = %v", seen)
	}
}
