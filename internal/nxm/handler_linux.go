package nxm

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	desktopID   = "tech.rethunk.Mortar.desktop"
	linuxAppID  = "tech.rethunk.Mortar"
	mortarMime  = "x-scheme-handler/mortar"
	fileMime    = "application/x-mortar"
	updateMIME  = "update-mime-database"
	updateIcons = "gtk4-update-icon-cache"
	xdgMime     = "/usr/bin/xdg-mime"
	updateDB    = "update-desktop-database"
	desktopPerm = 0o644
)

// packaged is set by `-X github.com/Rethunk-Tech/mortar/internal/nxm.packaged=` from the same PACKAGED
// value as main.packaged (deb, flatpak, …).
var packaged string

func skipUserDesktop() bool { return packaged != "" }

func inFlatpak() bool {
	if packaged == "flatpak" {
		return true
	}
	_, err := os.Stat("/.flatpak-info")
	return err == nil
}

// mimeDefaults runs xdg-mime. Inside a Flatpak the runtime has none, and the defaults that count are the host's.
func (l *System) mimeDefaults(args ...string) (string, error) { return l.onHost(xdgMime, args...) }

func schemeMime(scheme string) string { return "x-scheme-handler/" + scheme }

// schemeMimes is the MIME type of every URL scheme a source claims.
func schemeMimes() []string {
	var out []string
	for _, scheme := range source.Schemes() {
		out = append(out, schemeMime(scheme))
	}
	return out
}

// setDefaults makes Mortar the default app for every source scheme.
func (l *System) setDefaults() error {
	for _, mime := range schemeMimes() {
		if err := l.setDefault(mime); err != nil {
			return err
		}
	}
	return nil
}

func (l *System) setDefault(mime string) error {
	if _, err := l.mimeDefaults("default", l.handlerID(), mime); err != nil {
		return fmt.Errorf("xdg-mime default: %w", err)
	}
	return nil
}

// System is the system's registration of the source link schemes.
type System struct {
	exe  string
	home string
	// dataHome and configHome are the XDG base directories; dataDirs are the system ones, for reading names.
	dataHome, configHome string
	dataDirs             []string
	run                  Runner
}

// New returns the handler for this system, registering exe, or the AppImage file exe runs from.
func New(exe string) (*System, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	l := &System{
		exe: selfexe.Launchable(exe), home: home, dataHome: baseDir("XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
		configHome: baseDir("XDG_CONFIG_HOME", filepath.Join(home, ".config")), run: execRun,
	}
	for d := range strings.SplitSeq(baseDir("XDG_DATA_DIRS", "/usr/local/share:/usr/share"), ":") {
		l.dataDirs = append(l.dataDirs, filepath.Join(d, "applications"))
	}
	return l, nil
}

func baseDir(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return fallback
}

func execRun(name string, args ...string) (string, error) {
	out, err := exec.CommandContext(context.Background(), name, args...).Output()
	return string(out), err
}

func (l *System) desktopPath() string {
	return filepath.Join(l.dataHome, "applications", desktopID)
}

func (l *System) Owner(scheme string) (Owner, error) {
	out, err := l.mimeDefaults("query", "default", schemeMime(scheme))
	if err != nil {
		return Owner{}, fmt.Errorf("xdg-mime query: %w", err)
	}
	id := strings.TrimSpace(out)
	return Owner{ID: id, Name: l.appName(id), Mine: id != "" && id == l.handlerID()}, nil
}

// appName is the Name= of the desktop file, or the file's id without its extension.
func (l *System) appName(id string) string {
	if name, err := l.desktopEntryField(id, desktopName); err == nil {
		return name
	}
	return strings.TrimSuffix(id, ".desktop")
}

func desktopName(path string) string {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if name, ok := strings.CutPrefix(line, "Name="); ok {
			return name
		}
	}
	return ""
}

func desktopExec(path string) string {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if exec, ok := strings.CutPrefix(line, "Exec="); ok {
			return exec
		}
	}
	return ""
}

func (l *System) desktopEntryField(id string, read func(string) string) (string, error) {
	for _, dir := range append([]string{filepath.Join(l.dataHome, "applications")}, l.dataDirs...) {
		if v := read(filepath.Join(dir, id)); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("desktop entry %q not found", id)
}

// desktopFile is the user-level entry. It lists the source schemes only while Mortar handles it, so a system that
// picks a default from advertised types never picks Mortar once the user has switched it off.
func (l *System) desktopFile(withSchemes bool) string {
	mime := mortarMime + ";" + fileMime + ";"
	if withSchemes {
		for _, m := range slices.Backward(schemeMimes()) {
			mime = m + ";" + mime
		}
	}
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Mortar
Comment=Desktop mod manager for PC games
Exec="%s" %%u
Icon=mortar
Terminal=false
Categories=Game;Utility;
Keywords=mod;manager;nexus;stardew;
StartupWMClass=%s
MimeType=%s
`, quoteExec(l.exe), linuxAppID, mime)
}

// integrator is the id of an entry in the user's applications folder that an AppImage integrator (GearLever,
// AppImageLauncher, appimaged) wrote to run this executable on a link, or "". That entry is then the launcher and the
// link handler, and an entry of Mortar's own beside it would list Mortar twice for every scheme. An entry whose Exec
// sets DESKTOPINTEGRATION, the integrators' mark, wins over one that only names the executable.
func (l *System) integrator() string {
	if skipUserDesktop() {
		return ""
	}
	dir := filepath.Join(l.dataHome, "applications")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	found := ""
	for _, e := range entries {
		name := e.Name()
		// Mortar's own entry and its profile shortcuts all start with the app id.
		if !strings.HasSuffix(name, ".desktop") || strings.HasPrefix(name, linuxAppID) {
			continue
		}
		argv := splitDesktopExec(desktopExec(filepath.Join(dir, name)))
		marked := false
		if len(argv) > 0 && filepath.Base(argv[0]) == "env" {
			argv = argv[1:]
			for len(argv) > 0 && strings.Contains(argv[0], "=") {
				marked = marked || strings.HasPrefix(argv[0], "DESKTOPINTEGRATION=")
				argv = argv[1:]
			}
		}
		if len(argv) < 2 || argv[0] != l.exe || !slices.ContainsFunc(argv[1:], func(a string) bool { return a == "%u" || a == "%U" }) {
			continue
		}
		if marked {
			return name
		}
		found = cmp.Or(found, name)
	}
	return found
}

// handlerID is the desktop entry Mortar makes the default: an integrator's when there is one, else its own.
func (l *System) handlerID() string {
	if id := l.integrator(); id != "" {
		return id
	}
	return desktopID
}

// yieldTo removes Mortar's own entry in favour of the integrator's entry id and moves every default it held there.
func (l *System) yieldTo(id string) error {
	// Notifications still attach the icon.
	if err := l.installIcons(); err != nil {
		return err
	}
	path := l.desktopPath()
	if _, err := fsx.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := fsx.RemoveAll(path); err != nil {
		return err
	}
	_, _ = l.run(updateDB, filepath.Dir(path))
	for _, mime := range append(schemeMimes(), mortarMime, fileMime) {
		if out, err := l.mimeDefaults("query", "default", mime); err == nil && strings.TrimSpace(out) == desktopID {
			if _, err := l.mimeDefaults("default", id, mime); err != nil {
				return fmt.Errorf("xdg-mime default: %w", err)
			}
		}
	}
	return nil
}

func quoteExec(path string) string {
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '"' || c == '`' || c == '$' || c == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (l *System) iconPNGPath() string {
	return filepath.Join(l.dataHome, "icons", "hicolor", "256x256", "apps", "mortar.png")
}

func (l *System) iconSVGPath() string {
	return filepath.Join(l.dataHome, "icons", "hicolor", "scalable", "apps", "mortar.svg")
}

// NotificationIcon is the installed PNG, for notification attachments; empty until the desktop entry has been written.
func (l *System) NotificationIcon() string {
	path := l.iconPNGPath()
	if _, err := fsx.Stat(path); err != nil {
		return ""
	}
	return path
}

func (l *System) installIcons() error {
	png, svg := l.iconPNGPath(), l.iconSVGPath()
	if same(png, iconPNG) && same(svg, iconSVG) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(png), 0o750); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(svg), 0o750); err != nil {
		return err
	}
	if err := fsx.WriteFile(png, iconPNG, desktopPerm); err != nil {
		return err
	}
	if err := fsx.WriteFile(svg, iconSVG, desktopPerm); err != nil {
		return err
	}
	// GTK trusts an icon-theme.cache that is newer than the theme folder, and writing into its subfolders does not
	// touch the folder, so a cache another app left there would hide these icons.
	theme := filepath.Join(l.dataHome, "icons", "hicolor")
	now := time.Now()
	if err := os.Chtimes(theme, now, now); err != nil {
		return err
	}
	// A running shell may keep the cache it loaded; rebuilding it, when the tool exists, shows the icon at once.
	if _, err := fsx.Stat(filepath.Join(theme, "icon-theme.cache")); err == nil {
		_, _ = l.run(updateIcons, "-f", "-t", theme)
	}
	return nil
}

func (l *System) writeDesktop(withSchemes bool) error {
	if id := l.integrator(); id != "" {
		return l.yieldTo(id)
	}
	path := l.desktopPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if err := fsx.WriteFile(path, []byte(l.desktopFile(withSchemes)), desktopPerm); err != nil {
		return err
	}
	if err := l.installIcons(); err != nil {
		return err
	}
	// A missing tool or a failure only leaves the cache stale; xdg-mime reads the file itself.
	_, _ = l.run(updateDB, filepath.Dir(path))
	return nil
}

func (l *System) Register() error {
	if err := l.WriteNativeHosts(); err != nil {
		return err
	}
	if skipUserDesktop() {
		return l.setDefaults()
	}
	if err := l.writeDesktop(true); err != nil {
		return err
	}
	return l.setDefaults()
}

func (l *System) Restore(previous map[string]string) error {
	if err := l.removeNativeHosts(); err != nil {
		return err
	}
	if !skipUserDesktop() {
		if err := l.writeDesktop(false); err != nil {
			return err
		}
	}
	return l.handBack(source.Schemes(), previous)
}

// Release hands the given schemes back and rewrites Mortar's entry to list only the schemes still claimed.
func (l *System) Release(schemes []string, previous map[string]string) error {
	if !skipUserDesktop() {
		if err := l.writeDesktop(true); err != nil {
			return err
		}
	}
	return l.handBack(schemes, previous)
}

func (l *System) handBack(schemes []string, previous map[string]string) error {
	var unowned []string
	for _, scheme := range schemes {
		if previous[scheme] == "" {
			unowned = append(unowned, schemeMime(scheme))
			continue
		}
		if _, err := l.mimeDefaults("default", previous[scheme], schemeMime(scheme)); err != nil {
			return fmt.Errorf("xdg-mime default: %w", err)
		}
	}
	if len(unowned) == 0 {
		return nil
	}
	return l.dropDefault(unowned...)
}

// ForwardOther runs the previous handler's desktop entry on link.
func (l *System) ForwardOther(link, previous string) error {
	if inFlatpak() {
		if previous == "" {
			return errors.New("no previous nxm handler")
		}
		_, err := l.onHost("sh", "-c", launchOnHost, "sh", previous, link)
		return err
	}
	name, args, err := LinuxForwardArgv(previous, link, func(id string) (string, error) {
		return l.desktopEntryField(id, desktopExec)
	})
	if err != nil {
		return err
	}
	_, err = l.run(name, args...)
	return err
}

// dropDefault removes Mortar's lines for mimes from the user's mimeapps.list: xdg-mime cannot unset a default.
func (l *System) dropDefault(mimes ...string) error {
	config, err := l.hostConfigHome()
	if err != nil {
		return err
	}
	path := l.hostResolve(filepath.Join(config, "mimeapps.list"))
	b, err := l.hostRead(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	ids := []string{desktopID, l.handlerID()}
	kept := slices.DeleteFunc(slices.Clone(lines), func(s string) bool {
		k, v, ok := strings.Cut(strings.TrimSpace(s), "=")
		return ok && slices.Contains(mimes, k) && slices.Contains(ids, strings.TrimSuffix(v, ";"))
	})
	if len(kept) == len(lines) {
		return nil
	}
	return l.hostWrite(path, []byte(strings.Join(kept, "\n")))
}

const mimeXML = `<?xml version="1.0" encoding="UTF-8"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
  <mime-type type="application/x-mortar">
    <comment>Mortar profile</comment>
    <glob pattern="*.mortar"/>
    <sub-class-of type="application/zip"/>
  </mime-type>
</mime-info>
`

// RegisterLinks makes Mortar the app for mortar:// links and .mortar files: it writes the file type's MIME
// definition and the desktop entry, and sets both defaults. Running it again changes nothing. The nxm scheme is
// left as it is.
func (l *System) RegisterLinks() error {
	if skipUserDesktop() {
		for _, mime := range []string{mortarMime, fileMime} {
			if err := l.setDefault(mime); err != nil {
				return err
			}
		}
		return nil
	}
	xml := filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")
	if err := os.MkdirAll(filepath.Dir(xml), 0o750); err != nil {
		return err
	}
	if b, err := fsx.ReadFile(xml); err != nil || string(b) != mimeXML {
		if err := fsx.WriteFile(xml, []byte(mimeXML), desktopPerm); err != nil {
			return err
		}
		// A missing tool leaves the type unknown until the next database update; the desktop entry still works.
		_, _ = l.run(updateMIME, filepath.Join(l.dataHome, "mime"))
	}
	current, _ := fsx.ReadFile(l.desktopPath())
	if err := l.rewrite(current); err != nil {
		return err
	}
	for _, mime := range []string{mortarMime, fileMime} {
		if err := l.setDefault(mime); err != nil {
			return err
		}
	}
	return nil
}

// Refresh rewrites the desktop entry when Mortar has moved since it was written, as an AppImage does when the user
// moves the file. When no entry exists yet it registers mortar:// and .mortar so a skipped first run still gets
// those. Only a production build takes the entry, so a dev build or go run does not take it from the installed Mortar;
// the icons are any build's to bring up to date, since they name no program.
func (l *System) Refresh() error {
	if skipUserDesktop() {
		return nil
	}
	if _, err := fsx.Stat(l.desktopPath()); err == nil {
		if err := l.installIcons(); err != nil {
			return err
		}
	}
	if !production {
		return nil
	}
	return l.refresh()
}

// refresh leaves an entry whose Exec target still exists to that copy of Mortar: a second one running beside it,
// such as another AppImage, is not a move. When no entry exists it registers mortar:// and .mortar.
func (l *System) refresh() error {
	current, err := fsx.ReadFile(l.desktopPath())
	if errors.Is(err, fs.ErrNotExist) {
		return l.RegisterLinks()
	}
	if err != nil {
		return err
	}
	if target := execTarget(current); target != "" && target != l.exe {
		if _, err := os.Stat(target); err == nil {
			return nil
		}
	}
	return l.RegisterLinks()
}

func same(path string, want []byte) bool {
	b, err := fsx.ReadFile(path)
	return err == nil && bytes.Equal(b, want)
}

// execTarget is the program the desktop entry runs, as desktopFile writes it, or "" when it holds no such line.
func execTarget(entry []byte) string {
	for line := range strings.SplitSeq(string(entry), "\n") {
		if rest, ok := strings.CutPrefix(line, `Exec="`); ok {
			if target, _, ok := strings.Cut(rest, `"`); ok {
				return target
			}
		}
	}
	return ""
}

// rewrite writes the desktop entry unless current already is it, keeping the source schemes as current has them.
func (l *System) rewrite(current []byte) error {
	if id := l.integrator(); id != "" {
		return l.yieldTo(id)
	}
	mimes := schemeMimes()
	withSchemes := len(mimes) > 0 && strings.Contains(string(current), mimes[0])
	if string(current) == l.desktopFile(withSchemes) {
		return l.installIcons()
	}
	return l.writeDesktop(withSchemes)
}
