package game

import (
	"os"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testmain"
)

// Tests pass their own temporary home; a CI runner's XDG folders would otherwise win over it.
func TestMain(m *testing.M) {
	for _, k := range []string{"XDG_CONFIG_HOME", "HOST_XDG_CONFIG_HOME", "HOST_XDG_DATA_HOME"} {
		if err := os.Unsetenv(k); err != nil {
			panic(err)
		}
	}
	// The player's custom games load from the data folder, which a test must not point at the real one.
	testmain.RunInTempData(m, "game")
}
