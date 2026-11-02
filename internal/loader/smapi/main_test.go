package smapi

import (
	"os"
	"testing"
)

// Scratch folders are made under the data folder's cache, which a test must not point at the real one.
func TestMain(m *testing.M) {
	data, err := os.MkdirTemp("", "mortar-smapi-test-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", data)
	_ = os.Setenv("LOCALAPPDATA", data)
	code := m.Run()
	_ = os.RemoveAll(data)
	os.Exit(code)
}
