package main

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// minimisedArg starts Mortar in the tray for this launch only, whatever the start-minimised setting says, so a
// restart from a script or an update does not take the player's focus.
const minimisedArg = "--minimised"

// startedInTray reports whether args, the launch arguments after the program, ask to start in the tray.
func startedInTray(args []string) bool { return slices.Contains(args, minimisedArg) }

// trayWanted reports whether the settings, or a launch started in the tray, call for a tray icon: a hidden window
// with no icon could not be brought back.
func trayWanted(s settings.Settings, inTray bool) bool {
	return s.KeepInTray || s.StartMinimised || inTray
}
