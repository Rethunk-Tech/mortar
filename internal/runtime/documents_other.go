//go:build !windows

package runtime

import "path/filepath"

func documentsDir(home string) string { return filepath.Join(home, "Documents") }
