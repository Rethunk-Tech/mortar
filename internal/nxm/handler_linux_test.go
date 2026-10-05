package nxm

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

type recorder struct {
	calls   []string
	current string
	byMime  map[string]string
}

// run plays xdg-mime: query answers the current default, default changes it. Nothing touches the real system.
func (r *recorder) run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
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

func newLinux(t *testing.T, current string) (*System, *recorder) {
	t.Helper()
	dir := t.TempDir()
	r := &recorder{current: current, byMime: map[string]string{schemeMime("nxm"): current}}
	return &System{exe: "/opt/mortar/mortar", home: filepath.Join(dir, "home"), dataHome: filepath.Join(dir, "data"), configHome: filepath.Join(dir, "config"), run: r.run}, r
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
	png, err := fsx.ReadFile(l.iconPNGPath())
	if err != nil || len(png) == 0 || string(png) != string(iconPNG) {
		t.Fatalf("hicolor png: %d, %v", len(png), err)
	}
	svg, err := fsx.ReadFile(l.iconSVGPath())
	if err != nil || !strings.Contains(string(svg), `viewBox="0 0 1024 1024"`) {
		t.Fatalf("hicolor svg: %s, %v", svg, err)
	}
	if l.NotificationIcon() != l.iconPNGPath() {
		t.Fatalf("NotificationIcon: %q", l.NotificationIcon())
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
	if strings.Contains(string(b), schemeMime("nxm")) {
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
	target := filepath.Join(l.configHome, "dotfiles.list")
	if err := fsx.WriteFile(target, []byte("[Default Applications]\nx-scheme-handler/nxm="+desktopID+";\ntext/html=a.desktop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, list); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(""); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(list); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("mimeapps.list symlink replaced: %v", err)
	}
	b, _ := fsx.ReadFile(target)
	if string(b) != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("mimeapps.list: %q", b)
	}
	for _, c := range r.calls {
		if strings.HasPrefix(c, xdgMime+" default") && !strings.HasSuffix(c, desktopID+" "+schemeMime("nxm")) {
			t.Errorf("Restore set a default: %s", c)
		}
	}
}

func TestQuoteExec(t *testing.T) {
	path := `/opt/nexus mods/$app/mortar`
	got := quoteExec(path)
	if got != `/opt/nexus mods/\$app/mortar` {
		t.Fatalf("quoteExec = %q", got)
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
	if desktop, _ = fsx.ReadFile(l.desktopPath()); !strings.Contains(string(desktop), schemeMime("nxm")) {
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
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	img := filepath.Join(dir, "Mortar.AppImage")
	if err := fsx.WriteFile(img, nil, 0o700); err != nil {
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
	l.run = r.run
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if desktop, err := fsx.ReadFile(l.desktopPath()); err != nil || strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Fatalf("links entry before Register: %s, %v", desktop, err)
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	registered, _ := fsx.ReadFile(l.desktopPath())

	// Another copy of Mortar, such as a dev build, leaves the entry to the one it names.
	l.exe = filepath.Join(dir, "dev", "mortar")
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if desktop, _ := fsx.ReadFile(l.desktopPath()); string(desktop) != string(registered) {
		t.Errorf("another copy took the entry: %s", desktop)
	}

	l.exe = filepath.Join(dir, "Apps", "Mortar.AppImage")
	if err := os.Remove(img); err != nil {
		t.Fatal(err)
	}
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	desktop, _ := fsx.ReadFile(l.desktopPath())
	if !strings.Contains(string(desktop), `Exec="`+l.exe+`" %u`) || !strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Errorf("moved entry: %s", desktop)
	}
}

func TestRefreshRegistersLinksWhenMissing(t *testing.T) {
	l, r := newLinux(t, "vortex.desktop")
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	xml, err := fsx.ReadFile(filepath.Join(l.dataHome, "mime", "packages", "mortar.xml"))
	if err != nil || !strings.Contains(string(xml), `<glob pattern="*.mortar"/>`) {
		t.Fatalf("mime xml: %s, %v", xml, err)
	}
	desktop, err := fsx.ReadFile(l.desktopPath())
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
	l, _ := newLinux(t, "")
	other := filepath.Join(t.TempDir(), "other-mortar")
	if err := fsx.WriteFile(other, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	l.exe = other
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	registered, err := fsx.ReadFile(l.desktopPath())
	if err != nil {
		t.Fatal(err)
	}
	l.exe = filepath.Join(t.TempDir(), "this-mortar")
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	desktop, _ := fsx.ReadFile(l.desktopPath())
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

func TestRegisterLinksWritesReverseDNSDesktop(t *testing.T) {
	l, _ := newLinux(t, "")
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(l.desktopPath()) != desktopID {
		t.Fatalf("desktop path %s", l.desktopPath())
	}
	b, err := fsx.ReadFile(l.desktopPath())
	if err != nil || !strings.Contains(string(b), "StartupWMClass="+linuxAppID) {
		t.Fatalf("desktop file: %s, %v", b, err)
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

func TestPackagedRegisterSetsNxmDefaultWithoutWriting(t *testing.T) {
	packaged = "deb"
	t.Cleanup(func() { packaged = "" })
	l, r := newLinux(t, "vortex.desktop")
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if r.current != desktopID {
		t.Fatalf("default is %q after packaged Register", r.current)
	}
	if _, err := os.Stat(l.desktopPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("packaged Register wrote a user desktop entry")
	}
}

func TestFlatpakSkipsXdgMime(t *testing.T) {
	packaged = "flatpak"
	t.Cleanup(func() { packaged = "" })
	l, r := newLinux(t, "")
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if err := l.RegisterLinks(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Owner(); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 0 {
		t.Errorf("flatpak ran xdg-mime: %q", r.calls)
	}
	if _, err := os.Stat(l.desktopPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("flatpak wrote a user desktop entry")
	}
}

func TestRefreshUpdatesIconsWhenAnotherCopyOwnsTheEntry(t *testing.T) {
	l, _ := newLinux(t, "")
	other := filepath.Join(t.TempDir(), "other-mortar")
	if err := fsx.WriteFile(other, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	l.exe = other
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(l.iconSVGPath(), []byte("<svg/>"), desktopPerm); err != nil {
		t.Fatal(err)
	}
	l.exe = filepath.Join(t.TempDir(), "this-mortar")
	if err := l.Refresh(); err != nil {
		t.Fatal(err)
	}
	if svg, _ := fsx.ReadFile(l.iconSVGPath()); !bytes.Equal(svg, iconSVG) {
		t.Errorf("stale icon kept: %s", svg)
	}
}

func TestAnIntegratorsEntryHidesOurs(t *testing.T) {
	l, _ := newLinux(t, "")
	l.exe = "/home/u/Apps/mortar.appimage"
	apps := filepath.Join(l.dataHome, "applications")
	if err := os.MkdirAll(apps, 0o750); err != nil {
		t.Fatal(err)
	}
	// Our own profile shortcut and a hidden entry naming the same executable do not count as a launcher.
	shortcut := "[Desktop Entry]\nExec=\"" + l.exe + "\" --play=stardew/x\n"
	hiddenOther := "[Desktop Entry]\nExec=" + l.exe + "\nNoDisplay=true\n"
	for name, body := range map[string]string{linuxAppID + ".play-stardew-x.desktop": shortcut, "other.desktop": hiddenOther} {
		if err := fsx.WriteFile(filepath.Join(apps, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Register(); err != nil {
		t.Fatal(err)
	}
	if desktop, _ := fsx.ReadFile(l.desktopPath()); strings.Contains(string(desktop), "NoDisplay") {
		t.Fatalf("hidden without an integrator: %s", desktop)
	}

	gear := filepath.Join(apps, "mortar.desktop")
	if err := fsx.WriteFile(gear, []byte("[Desktop Entry]\nName=Mortar\nExec=env DESKTOPINTEGRATION=1 "+l.exe+" %u\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	desktop, _ := fsx.ReadFile(l.desktopPath())
	if !strings.Contains(string(desktop), "\nNoDisplay=true\n") || !strings.Contains(string(desktop), schemeMime("nxm")) {
		t.Fatalf("integrated entry: %s", desktop)
	}

	if err := os.Remove(gear); err != nil {
		t.Fatal(err)
	}
	if err := l.refresh(); err != nil {
		t.Fatal(err)
	}
	if desktop, _ := fsx.ReadFile(l.desktopPath()); strings.Contains(string(desktop), "NoDisplay") {
		t.Errorf("still hidden after the integrator's entry went: %s", desktop)
	}
}
