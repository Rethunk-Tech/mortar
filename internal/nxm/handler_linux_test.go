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
	if name == xdgMime && args[0] == "default" && args[2] == nxmMime {
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
	if err != nil || !strings.Contains(string(b), `Exec="/opt/mortar/mortar" %u`) || !strings.Contains(string(b), "MimeType=x-scheme-handler/nxm;x-scheme-handler/mortar;application/x-mortar;") {
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
	real := filepath.Join(l.configHome, "dotfiles.list")
	if err := fsx.WriteFile(real, []byte("[Default Applications]\nx-scheme-handler/nxm=mortar.desktop;\ntext/html=a.desktop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, list); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(""); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(list); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("mimeapps.list symlink replaced: %v", err)
	}
	b, _ := fsx.ReadFile(real)
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

func TestRegisterLinksIsIdempotentAndKeepsNxm(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	xml, err := fsx.ReadFile(filepath.Join(l.dataHome, "mime", "packages", "mortar.xml"))
	if err != nil || !strings.Contains(string(xml), `<glob pattern="*.mortar"/>`) {
		t.Fatalf("mime xml: %s, %v", xml, err)
	}
	desktop, _ := fsx.ReadFile(l.desktopPath())
	if strings.Contains(string(desktop), nxmMime) || !strings.Contains(string(desktop), "MimeType=x-scheme-handler/mortar;application/x-mortar;") {
		t.Fatalf("desktop file: %s", desktop)
	}
	if r.current != "vortex.desktop" {
		t.Errorf("RegisterLinks took the nxm default: %q", r.current)
	}
	want := []string{
		updateMIME + " " + filepath.Join(l.dataHome, "mime"),
		updateDB + " " + filepath.Dir(l.desktopPath()),
		xdgMime + " default " + desktopID + " " + mortarMime,
		xdgMime + " default " + desktopID + " " + fileMime,
	}
	if strings.Join(r.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls = %q", r.calls)
	}
	r.calls = nil
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 {
		t.Errorf("a second run only sets the defaults again, calls = %q", r.calls)
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if desktop, _ = fsx.ReadFile(l.desktopPath()); !strings.Contains(string(desktop), nxmMime) {
		t.Errorf("RegisterLinks dropped nxm: %s", desktop)
	}
}
