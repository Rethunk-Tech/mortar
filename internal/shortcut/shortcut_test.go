package shortcut

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseTakesOnlyAWellFormedPlayArgument(t *testing.T) {
	if r, ok := Parse([]string{"nxm://x", Arg("stardew", "abc123")}); !ok || r != (Request{Game: "stardew", Profile: "abc123"}) {
		t.Fatalf("got %+v %v", r, ok)
	}
	for _, bad := range []string{"--play=stardew", "--play=/abc", "--play=stardew/a b", `--play=stardew/a"b`, "--play=a/b/c"} {
		if _, ok := Parse([]string{bad}); ok {
			t.Errorf("%q must not parse", bad)
		}
	}
}

func TestReceiveKeepsTheRequestForTheWindowOnce(t *testing.T) {
	var events []string
	s := &Service{Emit: func(name string, _ any) { events = append(events, name) }}
	if s.Receive([]string{"--other"}) || s.Take() != nil {
		t.Fatal("no play argument, no request")
	}
	if !s.Receive([]string{Arg("stardew", "p1")}) || len(events) != 1 || events[0] != RequestedEvent {
		t.Fatalf("events %v", events)
	}
	if r := s.Take(); r == nil || r.Profile != "p1" {
		t.Fatalf("take %+v", r)
	}
	if s.Take() != nil {
		t.Fatal("a request is taken once")
	}
}

func TestCreateWritesADesktopEntryThatPlaysTheProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows shortcut goes through the shell")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path, err := create("/opt/Mortar/mortar", Arg("stardew", "p1"), "Main (Stardew Valley)")
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	b, err := root.ReadFile(filepath.Base(path))
	_ = root.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Name=Main (Stardew Valley)\n", `Exec="/opt/Mortar/mortar" --play=stardew/p1` + "\n", "Icon=mortar\n"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("entry lacks %q:\n%s", want, b)
		}
	}
}

func TestRenamedAndRemovedDesktopEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows shortcut goes through the shell")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path, err := create("/opt/Mortar/mortar", Arg("stardew", "p1"), "Main (Stardew Valley)")
	if err != nil {
		t.Fatal(err)
	}
	if err := Renamed("stardew", "p1", "Renamed", "Stardew Valley"); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	b, err := root.ReadFile(filepath.Base(path))
	_ = root.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Name=Renamed (Stardew Valley)\n") {
		t.Fatalf("renamed entry:\n%s", b)
	}
	if err := Removed("stardew", "p1"); err != nil {
		t.Fatal(err)
	}
	root, err = os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.ReadFile(filepath.Base(path))
	_ = root.Close()
	if !os.IsNotExist(err) {
		t.Fatalf("removed entry error = %v", err)
	}
	if err := Removed("stardew", "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestExistsAndRemoveDesktopEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows shortcut goes through the shell")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s := &Service{}
	if exists, err := s.Exists("stardew", "p1"); err != nil || exists {
		t.Fatalf("before create: exists=%v err=%v", exists, err)
	}
	if _, err := create("/opt/Mortar/mortar", Arg("stardew", "p1"), "Main"); err != nil {
		t.Fatal(err)
	}
	if exists, err := s.Exists("stardew", "p1"); err != nil || !exists {
		t.Fatalf("after create: exists=%v err=%v", exists, err)
	}
	if err := s.Remove("stardew", "p1"); err != nil {
		t.Fatal(err)
	}
	if exists, err := s.Exists("stardew", "p1"); err != nil || exists {
		t.Fatalf("after remove: exists=%v err=%v", exists, err)
	}
}
