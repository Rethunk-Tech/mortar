package nxm

import (
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
)

// RemoveDesktop undoes the user-level desktop integration for an uninstall: the desktop entry, its icons and the
// .mortar MIME definition, then the mortar:// and .mortar defaults. An entry that runs another Mortar copy which
// still exists (a second AppImage) is left with everything it uses. Run Restore first for the nxm default.
func (l *System) RemoveDesktop() error {
	if other, err := l.anotherCopysEntry(); err != nil || other {
		return err
	}
	if err := l.h.RemoveEntry(); err != nil {
		return err
	}
	for _, path := range []string{l.iconPNGPath(), l.iconSVGPath()} {
		if err := safefs.RemoveAll(path); err != nil {
			return err
		}
	}
	xml := filepath.Join(l.dataHome, "mime", "packages", "mortar.xml")
	if _, err := safefs.Stat(xml); err == nil {
		if err := safefs.RemoveAll(xml); err != nil {
			return err
		}
		_, _ = l.run(updateMIME, filepath.Join(l.dataHome, "mime"))
	}
	return l.h.DropDefault(mortarMime, fileMime)
}
