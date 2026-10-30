package nxm

import (
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
)

// RemoveDesktop undoes the user-level desktop integration for an uninstall: the desktop entry, its icons and the
// .mortar MIME definition, then the mortar:// and .mortar defaults. An entry that runs another Mortar copy which
// still exists (a second AppImage) is left with everything it uses. Run Restore first for the nxm default.
func (s *System) RemoveDesktop() error {
	if other, err := s.anotherCopysEntry(); err != nil || other {
		return err
	}
	if err := s.h.RemoveEntry(); err != nil {
		return err
	}
	for _, path := range []string{s.iconPNGPath(), s.iconSVGPath()} {
		if err := safefs.RemoveAll(path); err != nil {
			return err
		}
	}
	xml := filepath.Join(s.dataHome, "mime", "packages", "mortar.xml")
	if _, err := safefs.Stat(xml); err == nil {
		if err := safefs.RemoveAll(xml); err != nil {
			return err
		}
		_, _ = s.run(updateMIME, filepath.Join(s.dataHome, "mime"))
	}
	return s.h.DropDefault(mortarMime, fileMime)
}
