//go:build linux

package nxm

// A Flatpak build writes no host manifests (see WriteNativeHosts), so it has none to report.
func statusForExecutable(exe string) []HostStatus {
	if skipXdgMime() {
		return nil
	}
	l, err := New(exe)
	if err != nil {
		return nil
	}
	return l.NativeHostStatus()
}
