//go:build linux

package nxm

import (
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/nativehost"
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

func (l *System) nativeHostEntries() []browserEntry {
	var out []browserEntry
	for _, b := range nativeHostBrowsers {
		if b.firefox {
			if !fsx.IsDir(filepath.Join(l.home, b.dir)) {
				continue
			}
			out = append(out, b)
			continue
		}
		if !fsx.IsDir(filepath.Join(l.configHome, b.dir)) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func manifestPath(l *System, b browserEntry) string {
	if b.firefox {
		return filepath.Join(l.home, b.dir, "native-messaging-hosts", nativehost.Name+".json")
	}
	return filepath.Join(l.configHome, b.dir, "NativeMessagingHosts", nativehost.Name+".json")
}

// NativeHostStatus lists each installed browser's Mortar native-messaging manifest state.
func (l *System) NativeHostStatus() []HostStatus {
	var out []HostStatus
	for _, b := range l.nativeHostEntries() {
		path := manifestPath(l, b)
		out = append(out, HostStatus{
			Browser:      b.name,
			ManifestPath: path,
			State:        manifestState(l.exe, path),
		})
	}
	return out
}
