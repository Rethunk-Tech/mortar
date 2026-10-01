// Package datadir locates Mortar's per-user data folder.
package datadir

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// PointerName is the file in the default data folder that names a relocated data folder.
const PointerName = "data-location"

// Dir returns the user data folder, creating it (0700) if needed.
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

func resolve() (string, error) {
	def, err := defaultDir()
	if err != nil {
		return "", err
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
	return p, nil
}
