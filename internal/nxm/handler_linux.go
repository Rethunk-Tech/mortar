package nxm

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/schemehandler"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	desktopID   = appversion.AppID + ".desktop"
	mortarMime  = "x-scheme-handler/mortar"
	fileMime    = "application/x-mortar"
	updateMIME  = "update-mime-database"
	updateIcons = "gtk4-update-icon-cache"
	desktopPerm = 0o644
)

// packaged is set by `-X github.com/Rethunk-Tech/mortar/internal/nxm.packaged=` from the same PACKAGED
// value as main.packaged (deb, flatpak, …).
var packaged string

func skipUserDesktop() bool { return packaged != "" }

// New returns the handler for this system, registering exe, or the AppImage file exe runs from.
func New(exe string) (*System, error) { return newSystem(exe, execRun) }

func newSystem(exe string, run Runner) (*System, error) {
	exe = selfexe.Launchable(exe)
	h, err := schemehandler.New(schemehandler.Config{
		Exe:       exe,
		Schemes:   source.Schemes(),
		DesktopID: desktopID,
		Entry: schemehandler.Entry{
			Name:           "Mortar",
			Comment:        "Desktop mod manager for PC games",
			Icon:           "mortar",
			Categories:     []string{"Game", "Utility"},
			Keywords:       []string{"mod", "manager", "nexus", "stardew"},
			StartupWMClass: appversion.AppID,
			MimeTypes:      []string{mortarMime, fileMime},
		},
		Packaged: skipUserDesktop(),
		Run:      run,
	})
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	nh := nativehost.Host(exe)
	nh.Run = run
	return &System{exe: exe, dataHome: userdirs.DataHome(home), run: run, h: h, nh: nh}, nil
}

func execRun(name string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // G204: name is a desktop tool or the handler the user had
	cmd.Env = hostenv.Environ()
	out, err := cmd.Output()
	return string(out), err
}

func (l *System) desktopPath() string {
	return filepath.Join(l.dataHome, "applications", desktopID)
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
	if _, err := safefs.Stat(path); err != nil {
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
	if err := safefs.WriteFile(png, iconPNG, desktopPerm); err != nil {
		return err
	}
	if err := safefs.WriteFile(svg, iconSVG, desktopPerm); err != nil {
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
	if _, err := safefs.Stat(filepath.Join(theme, "icon-theme.cache")); err == nil {
		_, _ = l.run(updateIcons, "-f", "-t", theme)
	}
	return nil
}


// icons brings the icons up to date after a call that may have written the desktop entry, or yielded it to an
// AppImage integrator's: notifications still attach the icon then.
func (l *System) icons() error {
	if skipUserDesktop() {
		return nil
	}
	return l.installIcons()
}

func (l *System) Register() error {
	if err := l.WriteNativeHosts(); err != nil {
		return err
	}
	if err := l.h.Register(source.Schemes()...); err != nil {
		return err
	}
	return l.icons()
}

func (l *System) RegisterSchemes(schemes []string) error {
	if err := l.h.Register(schemes...); err != nil {
		return err
	}
	return l.icons()
}

func (l *System) Restore(previous map[string]string) error {
	if err := l.removeNativeHosts(); err != nil {
		return err
	}
	if err := l.h.Restore(previous); err != nil {
		return err
	}
	return l.icons()
}

// Release hands the given schemes back and leaves the rest claimed.
func (l *System) Release(schemes []string, previous map[string]string) error {
	if err := l.h.Release(schemes, previous); err != nil {
		return err
	}
	return l.icons()
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
	if !skipUserDesktop() {
		xml := filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")
		if err := os.MkdirAll(filepath.Dir(xml), 0o750); err != nil {
			return err
		}
		if b, err := safefs.ReadFile(xml); err != nil || string(b) != mimeXML {
			if err := safefs.WriteFile(xml, []byte(mimeXML), desktopPerm); err != nil {
				return err
			}
			// A missing tool leaves the type unknown until the next database update; the desktop entry still works.
			_, _ = l.run(updateMIME, filepath.Join(l.dataHome, "mime"))
		}
		if err := l.h.WriteEntry(); err != nil {
			return err
		}
		if err := l.installIcons(); err != nil {
			return err
		}
	}
	return l.h.SetDefault(mortarMime, fileMime)
}

// Refresh rewrites the desktop entry when Mortar has moved since it was written, as an AppImage does when the user
// moves the file. When no entry exists yet it registers mortar:// and .mortar so a skipped first run still gets
// those. Only a production build takes the entry, so a dev build or go run does not take it from the installed Mortar;
// the icons are any build's to bring up to date, since they name no program.
func (l *System) Refresh() error {
	if skipUserDesktop() {
		return nil
	}
	if _, err := safefs.Stat(l.desktopPath()); err == nil {
		if err := l.installIcons(); err != nil {
			return err
		}
	}
	if !production {
		return nil
	}
	return l.refresh()
}

// refresh leaves an entry that runs another copy of Mortar to that copy: a second one running beside it, such as
// another AppImage, is not a move. When no entry exists it registers mortar:// and .mortar.
func (l *System) refresh() error {
	other, err := l.anotherCopysEntry()
	if err != nil || other {
		return err
	}
	return l.RegisterLinks()
}

// anotherCopysEntry is whether the desktop entry runs a program other than this one that still exists.
func (l *System) anotherCopysEntry() (bool, error) {
	target, err := l.h.EntryProgram()
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if target == "" || target == l.exe {
		return false, nil
	}
	_, err = os.Stat(target)
	return err == nil, nil
}

func same(path string, want []byte) bool {
	b, err := safefs.ReadFile(path)
	return err == nil && bytes.Equal(b, want)
}
