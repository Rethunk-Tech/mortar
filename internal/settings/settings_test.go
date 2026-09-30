package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func open(t *testing.T) (*Store, string) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	return s, filepath.Dir(s.path)
}

func TestDefaultsAndRoundTrip(t *testing.T) {
	s, dir := open(t)
	if got := s.Get(); !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("defaults = %+v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.Accent, v.Translucent, v.LastGame = "moss", false, "lethal" }); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); !reflect.DeepEqual(got, Settings{"moss", false, "lethal", map[string]string{}, map[string]string{}, map[string]string{}}) {
		t.Fatalf("round trip = %+v", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != fileName {
		t.Fatalf("leftover files: %v", entries)
	}
	raw, _ := fsx.ReadFile(filepath.Join(dir, fileName))
	if want := `"lastGame": "lethal"`; !strings.Contains(string(raw), want) {
		t.Fatalf("json keys: %s", raw)
	}
}

func TestInvalidAccent(t *testing.T) {
	s, dir := open(t)
	if _, err := s.Update(func(v *Settings) { v.Accent = "neon" }); err == nil {
		t.Fatal("expected error")
	}
	if s.Get().Accent != "sand" {
		t.Fatal("state changed on rejected set")
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"neon","translucent":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open()
	if got := s2.Get(); got.Accent != "sand" || got.Translucent {
		t.Fatalf("load = %+v", got)
	}
}

func TestCorruptFile(t *testing.T) {
	s, dir := open(t)
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s2.Get(), Defaults()) {
		t.Fatalf("corrupt = %+v", s2.Get())
	}
	_ = s
}
