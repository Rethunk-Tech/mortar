package launchsvc

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

func TestBisectFailsOnlyOnACrash(t *testing.T) {
	if bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "These mods could not be added to your game."}}) {
		t.Fatal("a skipped mod is not a crash; counting it blames whichever mod is skipped")
	}
	if bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "Steam achievements won't work"}}) {
		t.Fatal("non-fatal Steam warning was treated as a crash")
	}
	if !bisectStartupFailure([]launch.Entry{{Level: launch.Alert, Message: "fatal startup error"}}) {
		t.Fatal("SMAPI crash report was not detected")
	}
	if !summaryHealthy(launch.Summary{Errors: 5}) {
		t.Fatal("a run with errors but no crash must count as healthy")
	}
	if summaryHealthy(launch.Summary{Crashed: true}) {
		t.Fatal("a crashed run must not count as healthy")
	}
}
