package nxm

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

// hostManifestPaths lists where each installed browser looks for Mortar's host manifest, with whether it is
// Firefox. A browser whose profile folder does not exist is skipped, so nothing is created for it.
func (l *System) hostManifestPaths() map[string]bool {
	paths := map[string]bool{}
	for _, b := range l.nativeHostEntries() {
		paths[manifestPath(l, b)] = b.firefox
	}
	return paths
}

// WriteNativeHosts lets the Mortar browser extension start this copy of Mortar. A browser outside the Flatpak
// sandbox cannot run the binary inside it, so a Flatpak build writes none.
func (l *System) WriteNativeHosts() error {
	if inFlatpak() {
		return nil
	}
	var errs []error
	for path, firefox := range l.hostManifestPaths() {
		m, err := nativehost.Manifest(l.exe, firefox)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(path), 0o700)
		}
		if err == nil {
			err = fsx.WriteFile(path, m, desktopPerm)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (l *System) removeNativeHosts() error {
	var errs []error
	for path := range l.hostManifestPaths() {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
