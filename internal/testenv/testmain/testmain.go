// Package testmain holds TestMain bodies shared by test packages; it imports no mortar package, so any test package may use it.
package testmain

import (
	"os"
	"testing"
)

// RunInTempData runs the package's tests with the data folder (XDG_DATA_HOME, LOCALAPPDATA) pointing at a temporary
// folder, so nothing a test writes under the data folder's cache reaches the real one, then exits with their result.
func RunInTempData(m *testing.M, name string) {
	data, err := os.MkdirTemp("", "mortar-"+name+"-test-")
	if err != nil {
		panic(err)
	}
	for _, k := range []string{"XDG_DATA_HOME", "LOCALAPPDATA"} {
		if err := os.Setenv(k, data); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(data)
	os.Exit(code)
}
