package nxm

import "github.com/Rethunk-Tech/go-desktopenv/nativemessaging"

// Host status values for NativeHostStatus rows.
const (
	HostOK         = string(nativemessaging.OK)
	HostMissing    = string(nativemessaging.Missing)
	HostStale      = string(nativemessaging.Stale)
	HostUnreadable = string(nativemessaging.Unreadable)
)

// HostStatus is one installed browser's native-messaging host manifest state.
type HostStatus struct {
	Browser      string `json:"browser"`
	ManifestPath string `json:"manifestPath"`
	State        string `json:"state"`
}

// RepairNativeHosts rewrites native-messaging host manifests for installed browsers.
func RepairNativeHosts(exe string) error {
	s, err := New(exe)
	if err != nil {
		return err
	}
	return s.WriteNativeHosts()
}

// StatusForExecutable reports native host manifests for browsers found on this system.
func StatusForExecutable(exe string) []HostStatus {
	s, err := New(exe)
	if err != nil {
		return nil
	}
	return s.NativeHostStatus()
}
