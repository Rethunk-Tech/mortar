package nxm

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// RemoveDesktop undoes the user-level desktop integration for an uninstall: the desktop entry, its icons and the
// .mortar MIME definition, then the mortar:// and .mortar defaults. An entry that runs another Mortar copy which
// still exists (a second AppImage) is left with everything it uses. Run Restore first for the nxm default.
func (l *System) RemoveDesktop() error {
	current, err := fsx.ReadFile(l.desktopPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	default:
		if target := execTarget(current); target != "" && target != l.exe {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		if err := fsx.RemoveAll(l.desktopPath()); err != nil {
			return err
		}
		_, _ = l.run(updateDB, filepath.Dir(l.desktopPath()))
	}
	for _, path := range []string{l.iconPNGPath(), l.iconSVGPath()} {
		if err := fsx.RemoveAll(path); err != nil {
			return err
		}
	}
	xml := filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")
	if _, err := fsx.Stat(xml); err == nil {
		if err := fsx.RemoveAll(xml); err != nil {
			return err
		}
		_, _ = l.run(updateMIME, filepath.Join(l.dataHome, "mime"))
	}
	return l.dropDefault(mortarMime, fileMime)
}
