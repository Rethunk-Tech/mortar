//go:build linux

package nxm

func repairNativeHosts(exe string) error {
	l, err := New(exe)
	if err != nil {
		return err
	}
	return l.WriteNativeHosts()
}
