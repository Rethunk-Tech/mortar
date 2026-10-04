// Package datadir locates Mortar's per-user data folder.
package datadir

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// PointerName is the file in the default data folder that names a relocated data folder.
const PointerName = "data-location"

// Dir returns the user data folder, creating it (0700) if needed: <exe dir>/data in portable mode, else the folder
// data-location names, else the OS default.
func Dir() (string, error) {
	dir, err := resolve()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return dir, nil
}

// DefaultDir is the OS default location, ignoring a relocation pointer.
func DefaultDir() (string, error) {
	return defaultDir()
}

func defaultDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "Mortar"), nil
	}
	if base := os.Getenv("XDG_DATA_HOME"); base != "" {
		return filepath.Join(base, "mortar"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	return filepath.Join(home, ".local", "share", "mortar"), nil
}

// errRealDataInTest stops a test from reading or writing the user's real Mortar data.
var errRealDataInTest = fmt.Errorf("tests must point XDG_DATA_HOME (LOCALAPPDATA on Windows) at a temporary folder")

func resolve() (string, error) {
	dir, err := portable()
	if err != nil {
		return "", err
	}
	if dir != "" {
		if testing.Testing() && !underTemp(dir) {
			return "", errRealDataInTest
		}
		return dir, nil
	}
	def, err := defaultDir()
	if err != nil {
		return "", err
	}
	if testing.Testing() && !underTemp(def) {
		return "", errRealDataInTest
	}
	b, err := fsx.ReadFile(filepath.Join(def, PointerName))
	if err != nil {
		if os.IsNotExist(err) {
			return def, nil
		}
		return "", err
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return def, nil
	}
	if info, err := os.Stat(p); err != nil || !info.IsDir() {
		return "", &MissingLocationError{Path: p}
	}
	return p, nil
}

// MissingLocationError says data-location names a folder that is not there (an unplugged drive, or a letter that now
// belongs to another volume). Creating it would start Mortar as a fresh install in the wrong place.
type MissingLocationError struct{ Path string }

func (e *MissingLocationError) Error() string {
	return fmt.Sprintf("the data folder %s cannot be found; its drive may be unplugged", e.Path)
}

// UseDefaultLocation drops the relocation pointer so Dir returns the OS default again.
func UseDefaultLocation() error {
	def, err := defaultDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(def, PointerName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SetLocation points Mortar at an existing folder that already holds (or will hold) its data.
func SetLocation(dir string) error {
	def, err := defaultDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(def, 0o700); err != nil {
		return err
	}
	return WriteFile(filepath.Join(def, PointerName), []byte(filepath.Clean(dir)+"\n"), 0o600)
}

// underTemp reports whether dir is inside a temp root; t.TempDir creates under GOTMPDIR when it is set.
func underTemp(dir string) bool {
	for _, root := range []string{os.TempDir(), os.Getenv("GOTMPDIR")} {
		if root == "" {
			continue
		}
		if UnderRoot(root, dir) {
			return true
		}
	}
	return false
}
