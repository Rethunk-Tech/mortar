//go:build windows

package settings

import (
	"errors"
	"os"

	winreg "golang.org/x/sys/windows/registry"
)

func applyAutostart(enable bool) error {
	key, _, err := winreg.CreateKey(winreg.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, winreg.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = key.Close() }()
	if !enable {
		err := key.DeleteValue("Mortar")
		if err != nil && !errors.Is(err, winreg.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue("Mortar", exe)
}
