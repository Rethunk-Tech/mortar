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
	if _, err := s.Update(func(v *Settings) {
		v.Accent, v.Background, v.BackgroundImage, v.LastGame = "moss", BackgroundSolid, "/pics/a.png", "lethal"
	}); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); !reflect.DeepEqual(got, Settings{"moss", BackgroundSolid, "/pics/a.png", "lethal", map[string]string{}, map[string]string{}, map[string]string{}}) {
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
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"neon","background":"neon"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open()
	if got := s2.Get(); got.Accent != "sand" || got.Background != BackgroundImage {
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

func TestBackground(t *testing.T) {
	s, dir := open(t)
	if s.Get().Background != BackgroundImage || s.Get().BackgroundImage != "" {
		t.Fatalf("defaults = %+v", s.Get())
	}
	if _, err := s.Update(func(v *Settings) { v.Background = "glass" }); err == nil {
		t.Fatal("expected error")
	}
	if s.Get().Background != BackgroundImage {
		t.Fatal("state changed on rejected set")
	}
	for _, b := range []string{BackgroundDesktop, BackgroundSolid, BackgroundImage} {
		if _, err := s.Update(func(v *Settings) { v.Background = b }); err != nil {
			t.Fatalf("%s: %v", b, err)
		}
	}
	// A file from before the field existed takes the defaults.
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"moss","translucent":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open()
	if got := s2.Get(); got.Accent != "moss" || got.Background != BackgroundImage {
		t.Fatalf("legacy file = %+v", got)
	}
}

func TestSetBackgroundImageValidates(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	svc.ValidateImage = func(p string) error {
		if p != "/ok.png" {
			return os.ErrInvalid
		}
		return nil
	}
	if err := svc.SetBackgroundImage("/etc/passwd"); err == nil || s.Get().BackgroundImage != "" {
		t.Fatalf("accepted a rejected path: %v %q", err, s.Get().BackgroundImage)
	}
	if err := svc.SetBackgroundImage("/ok.png"); err != nil || s.Get().BackgroundImage != "/ok.png" {
		t.Fatalf("set: %v %q", err, s.Get().BackgroundImage)
	}
	if err := svc.SetBackgroundImage(""); err != nil || s.Get().BackgroundImage != "" {
		t.Fatalf("reset: %v %q", err, s.Get().BackgroundImage)
	}
}
