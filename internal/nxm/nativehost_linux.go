package nxm

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/nativehost"
)

// chromiumConfigDirs are the Chromium browsers' user data directories under XDG_CONFIG_HOME; each reads host
// manifests from its own NativeMessagingHosts folder.
var chromiumConfigDirs = []string{
	"google-chrome", "google-chrome-beta", "google-chrome-unstable", "chromium",
	"BraveSoftware/Brave-Browser", "microsoft-edge", "vivaldi", "vivaldi-snapshot",
}

// hostManifestPaths lists where each installed browser looks for Mortar's host manifest, with whether it is
// Firefox. A browser whose profile folder does not exist is skipped, so nothing is created for it.
func (l *System) hostManifestPaths() map[string]bool {
	paths := map[string]bool{}
	for _, d := range chromiumConfigDirs {
		dir := filepath.Join(l.configHome, d)
		if _, err := os.Stat(dir); err == nil {
			paths[filepath.Join(dir, "NativeMessagingHosts", nativehost.Name+".json")] = false
		}
	}
	if dir := filepath.Join(l.home, ".mozilla"); dirExists(dir) {
		paths[filepath.Join(dir, "native-messaging-hosts", nativehost.Name+".json")] = true
	}
	return paths
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// WriteNativeHosts lets the Mortar browser extension start this copy of Mortar. A browser outside the Flatpak
// sandbox cannot run the binary inside it, so a Flatpak build writes none.
func (l *System) WriteNativeHosts() error {
	if skipXdgMime() {
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
