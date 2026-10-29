//go:build unix

package settings

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/portal"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
)

// RemoveAutostart deletes the autostart entry that starts Mortar at sign-in.
func RemoveAutostart() error { return applyAutostart(false) }

func applyAutostart(enable bool) error {
	if sandbox.InFlatpak() {
		// The sandbox cannot write the host's autostart folder; the Background portal writes the entry for it.
		return portal.SetAutostart(enable, []string{"mortar"})
	}
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
		body := autostartEntry(selfexe.Launchable(exe))
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

// autostartEntry is the autostart desktop entry for exe: the argument is quoted and escaped for the Exec value.
func autostartEntry(exe string) string {
	return "[Desktop Entry]\nType=Application\nName=Mortar\nExec=\"" + execArgEscaper.Replace(exe) + "\"\nX-GNOME-Autostart-enabled=true\n"
}

// execArgEscaper escapes a path for the inside of a quoted Exec argument: the quoted-argument escapes first, then the
// string-value escape that doubles each backslash, and %% so a percent is not read as a field code.
var execArgEscaper = strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`, `%`, `%%`)
