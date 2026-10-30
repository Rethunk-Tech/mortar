package nxm

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	xdgMime  = "/usr/bin/xdg-mime"
	updateDB = "update-desktop-database"
)

func schemeMime(scheme string) string { return "x-scheme-handler/" + scheme }

type recorder struct {
	calls   []string
	current string
	byMime  map[string]string
}

// run plays xdg-mime: query answers the current default, default changes it. Nothing touches the real system.
func (r *recorder) run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "flatpak-spawn" && len(args) > 2 && args[0] == "--host" && args[1] == "xdg-mime" {
		name, args = xdgMime, args[2:]
	}
	if name == xdgMime && args[0] == "query" {
		mime := args[len(args)-1]
		if r.byMime != nil {
			if v, ok := r.byMime[mime]; ok {
				return v + "\n", nil
			}
		}
		return r.current + "\n", nil
	}
	if name == xdgMime && args[0] == "default" {
		if r.byMime == nil {
			r.byMime = map[string]string{}
		}
		r.byMime[args[2]] = args[1]
		if args[2] == schemeMime("nxm") {
			r.current = args[1]
		}
	}
	return "", nil
}

// newLinux gives the test its own home and XDG folders and a recorder in place of xdg-mime.
func newLinux(t *testing.T, current string) (*System, *recorder) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "system"))
	r := &recorder{current: current, byMime: map[string]string{schemeMime("nxm"): current}}
	return withExe(t, r, "/opt/mortar/mortar"), r
}

// withExe is the System for another copy of Mortar at exe, in the same folders and with the same recorder.
func withExe(t *testing.T, r *recorder, exe string) *System {
	t.Helper()
	l, err := newSystem(exe, r.run)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestRegisterLinksIsIdempotentAndKeepsNxm(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	xml, err := safefs.ReadFile(filepath.Join(l.dataHome, "mime", "packages", "mortar.xml"))
	if err != nil || !strings.Contains(string(xml), `<glob pattern="*.mortar"/>`) {
		t.Fatalf("mime xml: %s, %v", xml, err)
	}
	desktop, _ := safefs.ReadFile(l.desktopPath())
	if strings.Contains(string(desktop), schemeMime("nxm")) || !strings.Contains(string(desktop), "MimeType=x-scheme-handler/mortar;application/x-mortar;") {
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
	if desktop, _ = safefs.ReadFile(l.desktopPath()); !strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Errorf("RegisterLinks dropped nxm: %s", desktop)
	}
}

func TestNotificationIconIsEmptyUntilTheDesktopEntryIsWritten(t *testing.T) {
	l, _ := newLinux(t, "")
	if l.NotificationIcon() != "" {
		t.Fatalf("icon before first run: %q", l.NotificationIcon())
	}
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if l.NotificationIcon() != l.iconPNGPath() {
		t.Fatalf("icon after RegisterLinks: %q", l.NotificationIcon())
	}
}

func TestAnAppImageRegistersItselfAndMovesItsEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	img := filepath.Join(dir, "Mortar.AppImage")
	if err := safefs.WriteFile(img, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", img)
	t.Setenv("APPDIR", "/tmp/.mount_Mortar")
	l, err := New("/tmp/.mount_Mortar/usr/bin/mortar")
	if err != nil || l.exe != img {
		t.Fatalf("exe %q, %v", l.exe, err)
	}
	if other, _ := New("/usr/bin/steam"); other.exe != "/usr/bin/steam" {
		t.Errorf("a child outside the mount took the AppImage path: %q", other.exe)
	}
	r := &recorder{}
	l = withExe(t, r, l.exe)
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if desktop, err := safefs.ReadFile(l.desktopPath()); err != nil || strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Fatalf("links entry before Register: %s, %v", desktop, err)
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	registered, _ := safefs.ReadFile(l.desktopPath())

	// Another copy of Mortar, such as a dev build, leaves the entry to the one it names.
	l = withExe(t, r, filepath.Join(dir, "dev", "mortar"))
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if desktop, _ := safefs.ReadFile(l.desktopPath()); string(desktop) != string(registered) {
		t.Errorf("another copy took the entry: %s", desktop)
	}

	l = withExe(t, r, filepath.Join(dir, "Apps", "Mortar.AppImage"))
	if err := os.Remove(img); err != nil {
		t.Fatal(err)
	}
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	desktop, _ := safefs.ReadFile(l.desktopPath())
	if !strings.Contains(string(desktop), `Exec="`+l.exe+`" %u`) || !strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Errorf("moved entry: %s", desktop)
	}
}

func TestRefreshRegistersLinksWhenMissing(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	xml, err := safefs.ReadFile(filepath.Join(l.dataHome, "mime", "packages", "mortar.xml"))
	if err != nil || !strings.Contains(string(xml), `<glob pattern="*.mortar"/>`) {
		t.Fatalf("mime xml: %s, %v", xml, err)
	}
	desktop, err := safefs.ReadFile(l.desktopPath())
	if err != nil || strings.Contains(string(desktop), schemeMime("nxm")) || !strings.Contains(string(desktop), "MimeType=x-scheme-handler/mortar;application/x-mortar;") {
		t.Fatalf("desktop file: %s, %v", desktop, err)
	}
	if r.current != "vortex.desktop" {
		t.Errorf("refresh took the nxm default: %q", r.current)
	}
	r.calls = nil
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 {
		t.Errorf("a second run only sets the defaults again, calls = %q", r.calls)
	}
}

func TestRefreshLeavesAnotherCopy(t *testing.T) {
	l, r := newLinux(t, "")
	other := filepath.Join(t.TempDir(), "other-mortar")
	if err := safefs.WriteFile(other, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	l = withExe(t, r, other)
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	registered, err := safefs.ReadFile(l.desktopPath())
	if err != nil {
		t.Fatal(err)
	}
	l = withExe(t, r, filepath.Join(t.TempDir(), "this-mortar"))
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	desktop, _ := safefs.ReadFile(l.desktopPath())
	if string(desktop) != string(registered) {
		t.Errorf("another copy took the entry: %s", desktop)
	}
}

func TestInstallIconsOutdatesAStaleIconCache(t *testing.T) {
	l, _ := newLinux(t, "")
	theme := filepath.Join(l.dataHome, "icons", "hicolor")
	if err := os.MkdirAll(theme, 0o750); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(theme, "icon-theme.cache")
	if err := os.WriteFile(cache, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(theme, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(cache, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := l.installIcons(); err != nil {
		t.Fatal(err)
	}
	dir, err := os.Stat(theme)
	if err != nil {
		t.Fatal(err)
	}
	c, err := os.Stat(cache)
	if err != nil {
		t.Fatal(err)
	}
	if dir.ModTime().Before(c.ModTime()) {
		t.Fatalf("theme folder %v is older than its cache %v, so GTK would keep the stale cache", dir.ModTime(), c.ModTime())
	}
}

func TestPackagedRegisterLinksSkipsUserDesktop(t *testing.T) {
	packaged = "deb"
	t.Cleanup(func() { packaged = "" })
	l, r := newLinux(t, "vortex.desktop")
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.desktopPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("packaged RegisterLinks wrote a user desktop entry")
	}
	if _, err := os.Stat(filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("packaged RegisterLinks wrote MIME XML")
	}
	if _, err := os.Stat(l.iconPNGPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("packaged RegisterLinks wrote icons")
	}
	want := []string{
		xdgMime + " default " + desktopID + " " + mortarMime,
		xdgMime + " default " + desktopID + " " + fileMime,
	}
	if strings.Join(r.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls = %q", r.calls)
	}
}

func TestPackagedRefreshSkipsUserDesktop(t *testing.T) {
	packaged = "deb"
	t.Cleanup(func() { packaged = "" })
	l, r := newLinux(t, "")
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.desktopPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("packaged refresh wrote a user desktop entry")
	}
	if len(r.calls) != 2 {
		t.Errorf("refresh still sets packaged mime defaults, calls = %q", r.calls)
	}
}

func TestRefreshUpdatesIconsWhenAnotherCopyOwnsTheEntry(t *testing.T) {
	l, r := newLinux(t, "")
	other := filepath.Join(t.TempDir(), "other-mortar")
	if err := safefs.WriteFile(other, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	l = withExe(t, r, other)
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(l.iconSVGPath(), []byte("<svg/>"), desktopPerm); err != nil {
		t.Fatal(err)
	}
	l = withExe(t, r, filepath.Join(t.TempDir(), "this-mortar"))
	if err := l.Refresh(); err != nil {
		t.Fatal(err)
	}
	if svg, _ := safefs.ReadFile(l.iconSVGPath()); !bytes.Equal(svg, iconSVG) {
		t.Errorf("stale icon kept: %s", svg)
	}
}

// writeApp writes a desktop entry into the user's applications folder.
func writeApp(t *testing.T, l *System, name, body string) {
	t.Helper()
	apps := filepath.Join(l.dataHome, "applications")
	if err := os.MkdirAll(apps, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(apps, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const gearLever = "mortar.appimage_0.desktop"

func gearLeverEntry(exe string) string {
	return "[Desktop Entry]\nName=Mortar\nExec=env DESKTOPINTEGRATION=1 " + exe + " %u\nX-AppImage-Name=Mortar\n"
}

func TestAnIntegratorsEntryIsTheHandler(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	l = withExe(t, r, "/home/u/Apps/mortar.appimage_0.appimage")
	writeApp(t, l, "appimagekit_other.desktop", "[Desktop Entry]\nExec=\""+l.exe+"\" %U\n")
	writeApp(t, l, gearLever, gearLeverEntry(l.exe))
	for range 2 {
		if err := l.Register(); err != nil {
			t.Fatal(err)
		}
		if err := l.RegisterLinks(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(l.desktopPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wrote its own entry beside the integrator's: %v", err)
	}
	for _, mime := range []string{schemeMime("nxm"), mortarMime, fileMime} {
		if r.byMime[mime] != gearLever {
			t.Errorf("%s default %q", mime, r.byMime[mime])
		}
	}
	if owner, _ := l.Owner("nxm"); !owner.Mine || owner.Name != "Mortar" {
		t.Errorf("owner: %+v", owner)
	}
	if l.NotificationIcon() == "" {
		t.Error("no notification icon")
	}
}


// The entry reads byte for byte as earlier releases wrote it, so an existing user entry is never rewritten.
func TestDesktopEntryMatchesTheOneEarlierReleasesWrote(t *testing.T) {
	l, r := newLinux(t, "")
	earlier := func(withSchemes bool) string {
		mime := ""
		if withSchemes {
			for _, scheme := range source.Schemes() {
				mime += "x-scheme-handler/" + scheme + ";"
			}
		}
		return `[Desktop Entry]
Type=Application
Name=Mortar
Comment=Desktop mod manager for PC games
Exec="/opt/mortar/mortar" %u
Icon=mortar
Terminal=false
Categories=Game;Utility;
Keywords=mod;manager;nexus;stardew;
StartupWMClass=tech.rethunk.Mortar
MimeType=` + mime + "x-scheme-handler/mortar;application/x-mortar;\n"
	}
	if err := os.MkdirAll(filepath.Dir(l.desktopPath()), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, withSchemes := range []bool{false, true} {
		if withSchemes {
			if err := l.Register(); err != nil {
				t.Fatal(err)
			}
		}
		if err := safefs.WriteFile(l.desktopPath(), []byte(earlier(withSchemes)), desktopPerm); err != nil {
			t.Fatal(err)
		}
		r.calls = nil
		if err := l.RegisterLinks(); err != nil {
			t.Fatal(err)
		}
		if slices.Contains(r.calls, updateDB+" "+filepath.Dir(l.desktopPath())) {
			t.Errorf("withSchemes %v: an earlier release's entry was rewritten", withSchemes)
		}
		if b, _ := safefs.ReadFile(l.desktopPath()); string(b) != earlier(withSchemes) {
			t.Errorf("withSchemes %v: entry\n%s\nwant\n%s", withSchemes, b, earlier(withSchemes))
		}
	}
}
