package nxm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestRemoveDesktopUndoesRegistration(t *testing.T) {
	l, _ := newLinux(t, "")
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(l.configHome, "mimeapps.list")
	if err := os.MkdirAll(l.configHome, 0o750); err != nil {
		t.Fatal(err)
	}
	body := "[Default Applications]\n" + mortarMime + "=" + desktopID + "\n" + fileMime + "=" + desktopID + ";\ntext/html=a.desktop\n"
	if err := fsx.WriteFile(list, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := l.RemoveDesktop(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{l.desktopPath(), l.iconPNGPath(), l.iconSVGPath(), filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still there: %v", p, err)
		}
	}
	if b, _ := fsx.ReadFile(list); string(b) != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("mimeapps.list: %q", b)
	}
}

func TestRemoveDesktopKeepsAnotherCopysEntry(t *testing.T) {
	l, _ := newLinux(t, "")
	other := filepath.Join(t.TempDir(), "Mortar.AppImage")
	if err := fsx.WriteFile(other, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	l.exe = other
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	l.exe = "/usr/bin/mortar"
	if err := l.RemoveDesktop(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.desktopPath()); err != nil {
		t.Errorf("another copy's entry was removed: %v", err)
	}
	if _, err := os.Stat(l.iconPNGPath()); err != nil {
		t.Errorf("another copy's icon was removed: %v", err)
	}
}
