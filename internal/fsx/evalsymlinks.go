package fsx

import "strings"

// extendedPath is the \\?\ form of an absolute Windows path of drive-letter form that is too long for the
// classic API, which is what filepath.EvalSymlinks calls on Windows without any prefixing of its own. Short, UNC and
// already-extended paths come back unchanged.
func extendedPath(abs string) string {
	if len(abs) < 248 || strings.HasPrefix(abs, `\\`) {
		return abs
	}
	return `\\?\` + abs
}
