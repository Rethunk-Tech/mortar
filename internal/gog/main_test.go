package gog

import (
	"os"
	"testing"
)

// Tests pass their own temporary home; a CI runner's XDG folders would otherwise win over it.
func TestMain(m *testing.M) {
	for _, k := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "HOST_XDG_CONFIG_HOME", "HOST_XDG_DATA_HOME"} {
		if err := os.Unsetenv(k); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}
