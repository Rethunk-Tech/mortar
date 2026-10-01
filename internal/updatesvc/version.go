package updatesvc

import (
	"os"
	"path/filepath"
	"strings"
)

const lastRunFile = "last-run-version"

func lastRunPath(dir string) string {
	return filepath.Join(dir, lastRunFile)
}

func readLastRun(dir string) (string, error) {
	b, err := os.ReadFile(lastRunPath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func writeLastRun(dir, version string) error {
	if dir == "" || version == "" {
		return nil
	}
	return os.WriteFile(lastRunPath(dir), []byte(version), 0o600)
}

// upgraded reports whether current is semver-newer than last.
func upgraded(current, last string) bool {
	if last == "" {
		return false
	}
	return versionNewer(current, last)
}
