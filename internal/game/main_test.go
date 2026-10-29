package game

import (
	"os"
	"testing"
)

// Tests pass their own temporary home; a CI runner's XDG folders would otherwise win over it.
func TestMain(m *testing.M) {
	for _, k := range []string{"XDG_CONFIG_HOME", "HOST_XDG_CONFIG_HOME", "HOST_XDG_DATA_HOME"} {
		if err := os.Unsetenv(k); err != nil {
			panic(err)
		}
	}
	// The player's custom games load from the data folder, which a test must not point at the real one.
	data, err := os.MkdirTemp("", "mortar-game-test-")
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
