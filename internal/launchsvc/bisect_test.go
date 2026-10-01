package launchsvc

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

func TestBisectStartupFailure(t *testing.T) {
	if !bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "These mods could not be added to your game."}}) {
		t.Fatal("missing-mod startup error was not detected")
	}
	if bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "Steam achievements won't work"}}) {
		t.Fatal("non-fatal Steam warning was treated as a startup failure")
	}
	if !bisectStartupFailure([]launch.Entry{{Level: launch.Alert, Message: "fatal startup error"}}) {
		t.Fatal("alert startup error was not detected")
	}
}
