package fsx

import "strings"

// ExtendedPath is the \\?\ form of an absolute Windows path of drive-letter form that is too long for the
// classic API, which is what filepath.EvalSymlinks and raw CreateFile calls use without any prefixing of their own. Short, UNC and
// already-extended paths come back unchanged.
func ExtendedPath(abs string) string {
	if len(abs) < 248 || strings.HasPrefix(abs, `\\`) {
		return abs
	}
	return `\\?\` + abs
}
