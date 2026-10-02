package nxm

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/nativehost"
)

// hostKeys are the HKCU keys, under software, whose default value points each browser at Mortar's host manifest,
// with whether it is Firefox. Brave and Vivaldi fall back to the Chromium and Chrome keys.
func hostKeys(software string) map[string]bool {
	return map[string]bool{
		software + `\Google\Chrome\NativeMessagingHosts\` + nativehost.Name:  false,
		software + `\Microsoft\Edge\NativeMessagingHosts\` + nativehost.Name: false,
		software + `\Chromium\NativeMessagingHosts\` + nativehost.Name:       false,
		software + `\Mozilla\NativeMessagingHosts\` + nativehost.Name:        true,
	}
}

// hostManifestDir holds the manifests in the default data folder, which stays local when the data folder moves.
func hostManifestDir() (string, error) {
	dir, err := datadir.DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "nativehost"), nil
}

func hostManifestPath(dir string, firefox bool) string {
	if firefox {
		return filepath.Join(dir, "firefox.json")
	}
	return filepath.Join(dir, "chromium.json")
}

// WriteNativeHosts lets the Mortar browser extension start this copy of Mortar. Keys are written for every browser,
// installed or not, so one installed later finds Mortar too.
func (w *System) WriteNativeHosts() error {
	dir, err := hostManifestDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, firefox := range []bool{false, true} {
		m, err := nativehost.Manifest(w.exe, firefox)
		if err != nil {
			return err
		}
		if err := fsx.WriteFile(hostManifestPath(dir, firefox), m, 0o600); err != nil {
			return err
		}
	}
	var errs []error
	for key, firefox := range hostKeys(w.software) {
		errs = append(errs, setDefault(key, hostManifestPath(dir, firefox)))
	}
	return errors.Join(errs...)
}

func setDefault(key, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", value)
}

func (w *System) removeNativeHosts() error {
	var errs []error
	for key := range hostKeys(w.software) {
		if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	dir, err := hostManifestDir()
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, firefox := range []bool{false, true} {
		if err := os.Remove(hostManifestPath(dir, firefox)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
