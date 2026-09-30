package updatesvc

import (
	"errors"
	"log"

	"golang.org/x/sys/windows/registry"
)

// uninstallKey is the per-user entry the NSIS installer writes (build/windows/nsis/wails_tools.nsh, UNINST_KEY).
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Rethunk.TechMortar`

// onApplied keeps Apps & features showing the running version after the updater replaced the installed exe. A
// portable copy has no entry, and none is created for it.
func onApplied(version string) func(string) {
	return func(string) {
		k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			return
		}
		if err == nil {
			err = k.SetStringValue("DisplayVersion", version)
			_ = k.Close()
		}
		if err != nil {
			log.Printf("update: record version %s: %v", version, err)
		}
	}
}
