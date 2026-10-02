package steam

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func writeGrid(dir string, appID uint32, source string) error {
	if source == "" || strings.Contains(source, "://") {
		return nil
	}
	ext := filepath.Ext(source)
	if ext == "" {
		return nil
	}
	body, err := fsx.ReadFile(source)
	if err != nil {
		return err
	}
	grid := filepath.Join(dir, "grid")
	if err := os.MkdirAll(grid, 0o750); err != nil {
		return err
	}
	base := strconv.FormatUint(uint64(appID), 10)
	for _, suffix := range []string{"p", "", "_hero"} {
		if err := fsx.WriteFile(filepath.Join(grid, base+suffix+ext), body, 0o600); err != nil {
			return err
		}
	}
	return nil
}
