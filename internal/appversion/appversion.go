// Package appversion reads Mortar's version from build/config.yml, the one place it is set.
package appversion

import (
	"errors"
	"regexp"
)

// infoVersion matches `version: "x"` indented under `info:`; the file's top-level `version: '3'` is the
// Taskfile schema version, not the app's.
var infoVersion = regexp.MustCompile(`(?m)^info:\n(?:[ \t]+.*\n)*?[ \t]+version:[ \t]*"([^"]+)"`)

// FromConfig returns info.version from the contents of build/config.yml.
func FromConfig(config []byte) (string, error) {
	m := infoVersion.FindSubmatch(config)
	if m == nil {
		return "", errors.New("build/config.yml has no info.version")
	}
	return string(m[1]), nil
}
