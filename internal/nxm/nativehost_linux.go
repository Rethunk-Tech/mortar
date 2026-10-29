package nxm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

// flatpakHostWrapper starts Mortar's native host from a browser outside the sandbox, which cannot run /app/bin/mortar
// itself; the browser's arguments pass through, so Mortar sees itself invoked as a native host.
const flatpakHostWrapper = "#!/bin/sh\nexec flatpak run --command=mortar " + appversion.AppID + " \"$@\"\n"

// nativeHostExe is what a host manifest runs: this executable, or inside a Flatpak the wrapper in the app's own
// folder under ~/.var/app, which the sandbox and the host see at the same path.
func (l *System) nativeHostExe() string {
	if flatpak.Inside() {
		return filepath.Join(userdirs.FlatpakApp(l.home, appversion.AppID), "native-host")
	}
	return l.exe
}

// WriteNativeHosts lets the Mortar browser extension start this copy of Mortar.
func (l *System) WriteNativeHosts() error {
	exe := l.nativeHostExe()
	if flatpak.Inside() {
		if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
			return err
		}
		if err := safefs.WriteFile(exe, []byte(flatpakHostWrapper), 0o700); err != nil {
			return err
		}
	}
	var errs []error
	for _, e := range l.nativeHostEntries() {
		m, err := nativehost.Manifest(exe, e.firefox)
		if err == nil {
			err = l.hostWrite(e.manifest, m)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (l *System) removeNativeHosts() error {
	var errs []error
	for _, e := range l.nativeHostEntries() {
		errs = append(errs, l.hostRemove(e.manifest))
	}
	return errors.Join(errs...)
}
