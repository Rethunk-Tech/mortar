package migrate

import (
	"os"
	"testing"
)

// The vortex reader unpacks under the data folder's cache, which a test must not point at the real one.
func TestMain(m *testing.M) {
	data, err := os.MkdirTemp("", "mortar-migrate-test-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", data)
	_ = os.Setenv("LOCALAPPDATA", data)
	code := m.Run()
	_ = os.RemoveAll(data)
	os.Exit(code)
}
