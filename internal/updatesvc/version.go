package updatesvc

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const lastRunFile = "last-run-version"

func lastRunPath(dir string) string {
	return filepath.Join(dir, lastRunFile)
}

func readLastRun(dir string) (string, error) {
	b, _, err := safefs.ReadFileIfExists(lastRunPath(dir))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func writeLastRun(dir, version string) error {
	if dir == "" || version == "" {
		return nil
	}
	return safefs.WriteFileAtomic(lastRunPath(dir), []byte(version), 0o600)
}

// upgraded reports whether current is semver-newer than last.
func upgraded(current, last string) bool {
	if last == "" {
		return false
	}
	return meta.Newer(current, last)
}
