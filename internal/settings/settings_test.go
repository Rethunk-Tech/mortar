package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/backup"
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
		v.NxmHandled, v.NxmPrevious = true, "vortex.desktop"
	}); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	want := s.Get()
	if got := s2.Get(); !reflect.DeepEqual(got, want) {
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
	if got := s2.CorruptPath(); got == "" {
		t.Fatal("corrupt settings path was not reported")
	} else if _, err := os.Stat(got); err != nil {
		t.Fatalf("corrupt copy missing: %v", err)
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

func TestSetGameFolderValidates(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	svc.ValidateGameFolder = func(_, dir string) error {
		if dir == "/bad" {
			return os.ErrInvalid
		}
		return nil
	}
	if err := svc.SetGameFolder("stardew", "/bad"); err == nil {
		t.Fatal("an invalid folder should be rejected")
	}
	if len(s.Get().GameFolders) != 0 {
		t.Fatal("a rejected folder must not be stored")
	}
	if err := svc.SetGameFolder("stardew", "/good"); err != nil || s.Get().GameFolders["stardew"] != "/good" {
		t.Fatalf("set: %v %v", err, s.Get().GameFolders)
	}
	if err := svc.SetGameFolder("stardew", ""); err != nil || len(s.Get().GameFolders) != 0 {
		t.Fatalf("clear: %v %v", err, s.Get().GameFolders)
	}
}

func TestBackupsKeptRange(t *testing.T) {
	s, dir := open(t)
	for _, n := range []int{MinBackupsKept - 1, MaxBackupsKept + 1} {
		if _, err := s.Update(func(v *Settings) { v.BackupsKept = n }); err == nil {
			t.Fatalf("BackupsKept %d accepted", n)
		}
	}
	if _, err := s.Update(func(v *Settings) { v.BackupsKept = MaxBackupsKept }); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"sand","background":"image","backupsKept":0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().BackupsKept; got != backup.DefaultKeep {
		t.Fatalf("out-of-range file loaded as %d, want %d", got, backup.DefaultKeep)
	}
}

func TestListColumns(t *testing.T) {
	s, dir := open(t)
	if got := s.Get(); !slices.Equal(got.ListColumns, defaultListColumns) || got.ListSortColumn != defaultListSortColumn || got.ListSortDir != defaultListSortDir || got.ListGroupBy != defaultListGroupBy {
		t.Fatalf("defaults = %+v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.ListColumns = []string{"on", "name", "nope"} }); err == nil {
		t.Fatal("unknown column accepted")
	}
	if !slices.Equal(s.Get().ListColumns, defaultListColumns) {
		t.Fatal("state changed on rejected set")
	}
	next := []string{"on", "name", "version"}
	if _, err := s.Update(func(v *Settings) {
		v.ListColumns, v.ListSortColumn, v.ListSortDir, v.ListGroupBy = next, "author", "desc", "tag"
	}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get(); !slices.Equal(got.ListColumns, next) || got.ListSortColumn != "author" || got.ListSortDir != "desc" || got.ListGroupBy != "tag" {
		t.Fatalf("set = %+v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.ListSortColumn = "nope" }); err == nil {
		t.Fatal("unknown sort column accepted")
	}
	if _, err := s.Update(func(v *Settings) { v.ListGroupBy = "status" }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(func(v *Settings) { v.ListGroupBy = "framework" }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(func(v *Settings) { v.ListGroupBy = "author" }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(func(v *Settings) { v.ListGroupBy = "nope" }); err == nil {
		t.Fatal("unknown group accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"sand","background":"image","listColumns":["nope","name"],"listSortColumn":"nope","listSortDir":"up","listGroupBy":"nope"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); !slices.Equal(got.ListColumns, []string{"on", "name"}) || got.ListSortColumn != defaultListSortColumn || got.ListSortDir != defaultListSortDir || got.ListGroupBy != defaultListGroupBy {
		t.Fatalf("load = %+v", got)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"accent":"sand","background":"image"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s3, _ := Open()
	if got := s3.Get(); !slices.Equal(got.ListColumns, defaultListColumns) {
		t.Fatalf("legacy = %v", got.ListColumns)
	}
}
