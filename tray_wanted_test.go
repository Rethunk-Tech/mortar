package main

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestTrayIsWantedByItsSettingsOrAStartInTheTray(t *testing.T) {
	for _, c := range []struct{ keep, minimised, inTray, want bool }{
		{false, false, false, false}, {true, false, false, true}, {false, true, false, true}, {false, false, true, true},
	} {
		if got := trayWanted(settings.Settings{KeepInTray: c.keep, StartMinimised: c.minimised}, c.inTray); got != c.want {
			t.Errorf("keep %v minimised %v in tray %v: %v", c.keep, c.minimised, c.inTray, got)
		}
	}
	if !startedInTray([]string{"--minimised"}) || startedInTray([]string{"nxm://x"}) {
		t.Error("only --minimised starts in the tray")
	}
}
