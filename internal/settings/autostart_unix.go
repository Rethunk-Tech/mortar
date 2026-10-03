//go:build unix

package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func applyAutostart(enable bool) error {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cfg = filepath.Join(home, ".config")
		if err := os.MkdirAll(cfg, 0o700); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if enable {
		if err := root.Mkdir("autostart", 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		sub, err := root.OpenRoot("autostart")
		if err != nil {
			return err
		}
		defer func() { _ = sub.Close() }()
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		body := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=Mortar\nExec=%s\nX-GNOME-Autostart-enabled=true\n", desktopExec(exe))
		return sub.WriteFile("mortar.desktop", []byte(body), 0o600)
	}
	sub, err := root.OpenRoot("autostart")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = sub.Close() }()
	err = sub.Remove("mortar.desktop")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func desktopExec(exe string) string {
	if strings.ContainsAny(exe, " \t") {
		return strconv.Quote(exe)
	}
	return exe
}
