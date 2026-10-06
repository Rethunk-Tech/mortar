//go:build linux

package nxm

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/nativehost"
)

type browserEntry struct {
	dir, name string
	firefox   bool
}

var nativeHostBrowsers = []browserEntry{
	{dir: "google-chrome", name: "Google Chrome"},
	{dir: "google-chrome-beta", name: "Google Chrome Beta"},
	{dir: "google-chrome-unstable", name: "Google Chrome Dev"},
	{dir: "chromium", name: "Chromium"},
	{dir: "BraveSoftware/Brave-Browser", name: "Brave"},
	{dir: "microsoft-edge", name: "Microsoft Edge"},
	{dir: "vivaldi", name: "Vivaldi"},
	{dir: "vivaldi-snapshot", name: "Vivaldi Snapshot"},
	{dir: ".mozilla", name: "Firefox", firefox: true},
}

// nativeHostEntries lists the installed browsers, each with where it looks for Mortar's host manifest. Chromium
// browsers look under the host's config folder, which inside a Flatpak is not the sandbox's own.
func (l *System) nativeHostEntries() []hostEntry {
	config, err := l.hostConfigHome()
	if err != nil {
		return nil
	}
	var out []hostEntry
	for _, b := range nativeHostBrowsers {
		dir := filepath.Join(config, b.dir)
		manifest := filepath.Join(dir, "NativeMessagingHosts", nativehost.Name+".json")
		if b.firefox {
			dir = filepath.Join(l.home, b.dir)
			manifest = filepath.Join(dir, "native-messaging-hosts", nativehost.Name+".json")
		}
		if l.hostIsDir(dir) {
			out = append(out, hostEntry{browserEntry: b, manifest: manifest})
		}
	}
	return out
}

type hostEntry struct {
	browserEntry
	manifest string
}

// NativeHostStatus lists each installed browser's Mortar native-messaging manifest state.
func (l *System) NativeHostStatus() []HostStatus {
	var out []HostStatus
	exe := l.nativeHostExe()
	for _, e := range l.nativeHostEntries() {
		b, err := l.hostRead(e.manifest)
		out = append(out, HostStatus{
			Browser:      e.name,
			ManifestPath: e.manifest,
			State:        manifestStateOf(exe, b, err),
		})
	}
	return out
}
