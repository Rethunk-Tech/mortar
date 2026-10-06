//go:build linux

package nxm

func statusForExecutable(exe string) []HostStatus {
	l, err := New(exe)
	if err != nil {
		return nil
	}
	return l.NativeHostStatus()
}
