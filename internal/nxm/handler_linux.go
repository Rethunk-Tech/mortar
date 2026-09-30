package nxm

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	desktopID   = "mortar.desktop"
	nxmMime     = "x-scheme-handler/nxm"
	mortarMime  = "x-scheme-handler/mortar"
	fileMime    = "application/x-mortar"
	updateMIME  = "update-mime-database"
	xdgMime     = "/usr/bin/xdg-mime"
	updateDB    = "update-desktop-database"
	desktopPerm = 0o644
)

// System is the system's registration of the nxm scheme.
type System struct {
	exe string
	// dataHome and configHome are the XDG base directories; dataDirs are the system ones, for reading names.
	dataHome, configHome string
	dataDirs             []string
	run                  Runner
}

// New returns the handler for this system, registering exe.
func New(exe string) (*System, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	l := &System{
		exe: exe, dataHome: baseDir("XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
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

func (l *System) desktopPath() string { return filepath.Join(l.dataHome, "applications", desktopID) }

func (l *System) Owner() (Owner, error) {
	out, err := l.run(xdgMime, "query", "default", nxmMime)
	if err != nil {
		return Owner{}, fmt.Errorf("xdg-mime query: %w", err)
	}
	id := strings.TrimSpace(out)
	return Owner{ID: id, Name: l.appName(id), Mine: id == desktopID}, nil
}

// appName is the Name= of the desktop file, or the file's id without its extension.
func (l *System) appName(id string) string {
	for _, dir := range append([]string{filepath.Join(l.dataHome, "applications")}, l.dataDirs...) {
		if name := desktopName(filepath.Join(dir, id)); name != "" {
			return name
		}
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

// desktopFile is the user-level entry. It lists the nxm scheme only while Mortar handles it, so a system that
// picks a default from advertised types never picks Mortar once the user has switched it off.
func (l *System) desktopFile(withNxm bool) string {
	mime := mortarMime + ";" + fileMime + ";"
	if withNxm {
		mime = nxmMime + ";" + mime
	}
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Mortar
Comment=Multi-game desktop mod manager
Exec="%s" %%u
Icon=mortar
Terminal=false
Categories=Game;Utility;
StartupWMClass=mortar
MimeType=%s
`, l.exe, mime)
}

func (l *System) writeDesktop(withNxm bool) error {
	if strings.ContainsAny(l.exe, "\"\\`$%") {
		return fmt.Errorf("cannot register %s: its path has a character a desktop file cannot quote", l.exe)
	}
	path := l.desktopPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if err := fsx.WriteFile(path, []byte(l.desktopFile(withNxm)), desktopPerm); err != nil {
		return err
	}
	// A missing tool or a failure only leaves the cache stale; xdg-mime reads the file itself.
	_, _ = l.run(updateDB, filepath.Dir(path))
	return nil
}

func (l *System) Register() error {
	if err := l.writeDesktop(true); err != nil {
		return err
	}
	if _, err := l.run(xdgMime, "default", desktopID, nxmMime); err != nil {
		return fmt.Errorf("xdg-mime default: %w", err)
	}
	return nil
}

func (l *System) Restore(previous string) error {
	if err := l.writeDesktop(false); err != nil {
		return err
	}
	if previous != "" {
		if _, err := l.run(xdgMime, "default", previous, nxmMime); err != nil {
			return fmt.Errorf("xdg-mime default: %w", err)
		}
		return nil
	}
	return l.dropDefault()
}

// dropDefault removes Mortar's line from the user's mimeapps.list: xdg-mime cannot unset a default.
func (l *System) dropDefault() error {
	path := filepath.Join(l.configHome, "mimeapps.list")
	// A dotfile manager's symlink is followed, so the rewrite lands in its target and the link stays.
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
	b, err := fsx.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	kept := slices.DeleteFunc(slices.Clone(lines), func(s string) bool {
		k, v, ok := strings.Cut(strings.TrimSpace(s), "=")
		return ok && k == nxmMime && strings.TrimSuffix(v, ";") == desktopID
	})
	if len(kept) == len(lines) {
		return nil
	}
	return datadir.WriteFile(path, []byte(strings.Join(kept, "\n")), desktopPerm)
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
	withNxm := strings.Contains(string(current), nxmMime)
	if string(current) != l.desktopFile(withNxm) {
		if err := l.writeDesktop(withNxm); err != nil {
			return err
		}
	}
	for _, mime := range []string{mortarMime, fileMime} {
		if _, err := l.run(xdgMime, "default", desktopID, mime); err != nil {
			return fmt.Errorf("xdg-mime default: %w", err)
		}
	}
	return nil
}
