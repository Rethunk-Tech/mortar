//go:build windows

package nxm

func repairNativeHosts(exe string) error {
	w, err := New(exe)
	if err != nil {
		return err
	}
	return w.WriteNativeHosts()
}
