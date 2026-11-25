package tools

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	itemstore "github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestNewService(t *testing.T) {
	if _, err := NewService(t.TempDir(), nil, nil); err == nil {
		t.Fatal("expected Open to refuse the real data dir")
	}
	testfs.DataHome(t)
	s, err := NewService(t.TempDir(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.store == nil {
		t.Fatal("nil store")
	}
}

func TestServiceListLaunchRemove(t *testing.T) {
	s := &Service{store: &Store{root: t.TempDir()}}
	got, err := s.List("stardew", "")
	if err != nil || len(got) != 0 {
		t.Fatalf("List = %#v, %v", got, err)
	}
	if err := s.Launch("nope", "p", "id"); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("Launch unknown game: %v", err)
	}
	if err := s.Launch("stardew", "p", "missing"); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("Launch missing tool: %v", err)
	}
	if err := s.Remove("stardew", "missing"); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("Remove: %v", err)
	}
}

func TestAddUpdateValidation(t *testing.T) {
	st, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{store: &Store{root: t.TempDir()}, home: t.TempDir(), settings: st}
	if _, err := s.Add("nope", Tool{Name: "n", Executable: "/bin/true"}); err == nil {
		t.Fatal("unknown game")
	}
	if _, err := s.Add("stardew", Tool{}); err == nil {
		t.Fatal("empty tool")
	}
	if err := s.Update("nope", Tool{ID: "x", Name: "n", Executable: "/bin/true"}); err == nil {
		t.Fatal("unknown game")
	}
	if err := s.Update("stardew", Tool{Name: "n"}); err == nil {
		t.Fatal("missing executable")
	}
}

func TestAddThenLaunchWithoutInstall(t *testing.T) {
	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true in PATH")
	}
	dir := t.TempDir()
	st, err := settings.OpenIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{
		store:    &Store{root: t.TempDir()},
		home:     t.TempDir(),
		settings: st,
		profiles: profile.OpenIn(dir, itemstore.OpenAt(filepath.Join(dir, "items"))),
	}
	tool, err := s.Add("stardew", Tool{Name: "true", Executable: truePath, Arguments: []string{"--noop"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update("stardew", tool); err != nil {
		t.Fatal(err)
	}
	if err := s.Launch("stardew", "p1", tool.ID); err == nil {
		t.Fatal("expected not installed")
	}
}

func TestSavesDir(t *testing.T) {
	st, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{home: t.TempDir(), settings: st}
	dir, err := s.savesDir("nope", "")
	if err != nil || dir != "" {
		t.Fatalf("no saves = %q, %v", dir, err)
	}
	if _, err := s.savesDir("stardew", ""); err != nil {
		t.Fatal(err)
	}
}

func TestContextErrors(t *testing.T) {
	dir := t.TempDir()
	st, err := settings.OpenIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{
		home:     t.TempDir(),
		settings: st,
		profiles: profile.OpenIn(dir, itemstore.OpenAt(filepath.Join(dir, "items"))),
	}
	if _, err := s.context("nope", "p"); err == nil {
		t.Fatal("unknown game")
	}
	gameDir := t.TempDir()
	if _, err := st.Update(func(v *settings.Settings) {
		v.GameFolders = map[string]string{"stardew": gameDir}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.context("stardew", "missing"); err == nil {
		t.Fatal("expected missing profile")
	}
}
