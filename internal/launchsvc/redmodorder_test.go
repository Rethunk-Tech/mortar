package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A launch whose archive order cannot be read stops, rather than loading the order the last launch wrote.
func TestALaunchStopsWhenTheArchiveOrderCannotBeRead(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, profiles := testenv.Stores(t)
	svc := NewService(t.TempDir(), nil, profiles)
	if err := svc.deployREDmods(t.Context(), "cyberpunk-2077", game.Install{Dir: t.TempDir()}, "no-such-profile", nil, game.StartEnv{}); err == nil {
		t.Fatal("the launch went on without its archive order")
	}
}
