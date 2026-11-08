package devsvc

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-editorlocator"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestAsEditorsKeepsFrontendJSON(t *testing.T) {
	got := asEditors([]editorlocator.Editor{{
		Name:    "VS Code",
		Program: "/usr/bin/code",
		Kind:    editorlocator.VSCodeFamily,
	}})
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `[{"name":"VS Code","path":"/usr/bin/code"}]` {
		t.Fatalf("got %s", raw)
	}
}

func TestAChosenEditorThatIsGoneSaysSo(t *testing.T) {
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "gone", "rider64.exe")
	if _, err := store.Update(func(s *settings.Settings) { s.CodeEditor = gone }); err != nil {
		t.Fatal(err)
	}
	s := &Service{Settings: settings.NewService(store)}
	err = s.openIn("unused", "unused", func(string) []string { return nil })
	if err == nil || !strings.Contains(err.Error(), "rider64.exe") {
		t.Fatalf("got %v", err)
	}
}
