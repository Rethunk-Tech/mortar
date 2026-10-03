//go:build windows

package nxm

import (
	"errors"

	"golang.org/x/sys/windows/registry"

	"github.com/Rethunk-AI/mortar/internal/nativehost"
)

type winHost struct {
	name   string
	probe  string
	regKey string
}

func (w *System) nativeHostRows() []winHost {
	sw := w.software
	host := `\NativeMessagingHosts\` + nativehost.Name
	return []winHost{
		{name: "Google Chrome", probe: sw + `\Google\Chrome`, regKey: sw + `\Google\Chrome` + host},
		{name: "Microsoft Edge", probe: sw + `\Microsoft\Edge`, regKey: sw + `\Microsoft\Edge` + host},
		{name: "Chromium", probe: sw + `\Chromium`, regKey: sw + `\Chromium` + host},
		{name: "Firefox", probe: sw + `\Mozilla`, regKey: sw + `\Mozilla` + host},
	}
}

// NativeHostStatus lists each installed browser's Mortar native-messaging manifest state.
func (w *System) NativeHostStatus() []HostStatus {
	var out []HostStatus
	for _, b := range w.nativeHostRows() {
		if !w.browserInstalled(b.probe) {
			continue
		}
		path, state := w.registryManifestState(b.regKey)
		out = append(out, HostStatus{Browser: b.name, ManifestPath: path, State: state})
	}
	return out
}

func (w *System) browserInstalled(probe string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, probe, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	_ = k.Close()
	return true
}

func (w *System) registryManifestState(hostKey string) (manifestPath, state string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, hostKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return hostKey, HostMissing
	}
	if err != nil {
		return hostKey, HostUnreadable
	}
	path, _, err := k.GetStringValue("")
	_ = k.Close()
	if err != nil {
		return hostKey, HostUnreadable
	}
	return path, manifestState(w.exe, path)
}

func statusForExecutable(exe string) []HostStatus {
	w, err := New(exe)
	if err != nil {
		return nil
	}
	return w.NativeHostStatus()
}
