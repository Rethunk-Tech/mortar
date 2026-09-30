package nxm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

type recorder struct {
	calls   []string
	current string
}

// run plays xdg-mime: query answers the current default, default changes it. Nothing touches the real system.
func (r *recorder) run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == xdgMime && args[0] == "query" {
		return r.current + "\n", nil
	}
	if name == xdgMime && args[0] == "default" {
		r.current = args[1]
	}
	return "", nil
}

func newLinux(t *testing.T, current string) (*System, *recorder) {
	t.Helper()
	dir := t.TempDir()
	r := &recorder{current: current}
	return &System{exe: "/opt/mortar/mortar", dataHome: filepath.Join(dir, "data"), configHome: filepath.Join(dir, "config"), run: r.run}, r
}

func TestRegisterThenRestorePreviousHandler(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	owner, err := l.Owner()
	if err != nil || owner.ID != "vortex.desktop" || owner.Mine || owner.Name != "vortex" {
		t.Fatalf("owner before: %+v, %v", owner, err)
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if r.current != desktopID {
		t.Fatalf("default is %q after Register", r.current)
	}
	b, err := fsx.ReadFile(l.desktopPath())
	if err != nil || !strings.Contains(string(b), `Exec="/opt/mortar/mortar" %u`) || !strings.Contains(string(b), "MimeType=x-scheme-handler/nxm;x-scheme-handler/mortar;") {
		t.Fatalf("desktop file: %s, %v", b, err)
	}
	if owner, _ := l.Owner(); !owner.Mine {
		t.Fatalf("owner after Register: %+v", owner)
	}
	if err := l.Restore("vortex.desktop"); err != nil {
		t.Fatal(err)
	}
	if r.current != "vortex.desktop" {
		t.Errorf("default is %q after Restore", r.current)
	}
	b, _ = fsx.ReadFile(l.desktopPath())
	if strings.Contains(string(b), nxmMime) {
		t.Errorf("desktop file still lists nxm after Restore: %s", b)
	}
}

func TestRestoreWithoutPreviousDropsOurDefault(t *testing.T) {
	l, r := newLinux(t, "")
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(l.configHome, "mimeapps.list")
	if err := os.MkdirAll(l.configHome, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(list, []byte("[Default Applications]\nx-scheme-handler/nxm=mortar.desktop\ntext/html=a.desktop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(""); err != nil {
		t.Fatal(err)
	}
	b, _ := fsx.ReadFile(list)
	if string(b) != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("mimeapps.list: %q", b)
	}
	for _, c := range r.calls {
		if strings.HasPrefix(c, xdgMime+" default") && !strings.HasSuffix(c, desktopID+" "+nxmMime) {
			t.Errorf("Restore set a default: %s", c)
		}
	}
}

func TestRegisterRefusesAPathItCannotQuote(t *testing.T) {
	l, r := newLinux(t, "")
	l.exe = `/opt/mor"tar`
	if err := l.Register(); err == nil || len(r.calls) != 0 {
		t.Errorf("err %v, calls %v", err, r.calls)
	}
}
