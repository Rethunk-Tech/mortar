// Package datadirtest points a test's data folder at a temporary one on every OS.
package datadirtest

import "testing"

// Use makes datadir resolve under base: XDG_DATA_HOME on Linux, LOCALAPPDATA on Windows.
func Use(t testing.TB, base string) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("LOCALAPPDATA", base)
}
