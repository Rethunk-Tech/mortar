package settings

import "testing"

func TestResolveOrder(t *testing.T) {
	t.Parallel()
	var empty Settings
	if got := ResolveAt(empty, "defaultLaunchMethod", Scope{Game: "stardew"}, nil); got != LaunchDirect {
		t.Fatalf("default = %q", got)
	}
	game := Settings{}
	putGame(&game, "stardew", GameSettings{DefaultLaunchMethod: LaunchSteam})
	if got := ResolveAt(game, "defaultLaunchMethod", Scope{Game: "stardew"}, nil); got != LaunchSteam {
		t.Fatalf("game = %q", got)
	}
	if got := ResolveAt(game, "defaultLaunchMethod", Scope{Game: "stardew"}, map[string]string{"defaultLaunchMethod": LaunchDirect}); got != LaunchDirect {
		t.Fatalf("profile = %q", got)
	}
}

func TestResolveSkipPlayCheck(t *testing.T) {
	t.Parallel()
	var empty Settings
	if got := ResolveAt(empty, "skipPlayCheck", Scope{Game: "stardew"}, nil); got != "false" {
		t.Fatalf("default skip = %q", got)
	}
	game := Settings{}
	putGame(&game, "stardew", GameSettings{SkipPlayCheck: true})
	if got := ResolveAt(game, "skipPlayCheck", Scope{Game: "stardew"}, nil); got != "true" {
		t.Fatalf("game skip = %q", got)
	}
	if got := ResolveAt(game, "skipPlayCheck", Scope{Game: "stardew"}, map[string]string{"skipPlayCheck": "false"}); got != "false" {
		t.Fatalf("profile skip = %q", got)
	}
}

func TestProfileOverridableKeys(t *testing.T) {
	t.Parallel()
	for _, key := range []string{
		"defaultLaunchMethod", "showSmapiConsole", "backupBeforePlay",
		"saveBackupsKept", "updateModsBeforePlayDefault", "skipPlayCheck", "skipIntro",
		"saveBackupHours", "saveBackupKeep", "runsKept", "consoleLogCap", "consoleLevel", "consoleTimestamps",
		"consoleFollow", "cosmeticConflicts", "conflictScanDepth", "enableRequirements", "missingRequirements",
	} {
		if !ProfileOverridable(key) {
			t.Fatalf("%s should be profile-overridable", key)
		}
	}
	if ProfileOverridable("oldFilesOnUpdate") {
		t.Fatal("oldFilesOnUpdate must not be profile-overridable")
	}
}

func TestGraphicsApiIsUnsetByDefaultAndAProfileMayOverrideIt(t *testing.T) {
	var s Settings
	if got := ResolveAt(s, "graphicsApi", Scope{Game: "peak"}, nil); got != "" {
		t.Fatalf("default = %q, want unset", got)
	}
	if err := ApplyKeyGame(&s, "graphicsApi", "dx12", "peak"); err != nil {
		t.Fatal(err)
	}
	if got := ResolveAt(s, "graphicsApi", Scope{Game: "peak"}, nil); got != "dx12" {
		t.Errorf("game = %q", got)
	}
	if got := ResolveAt(s, "graphicsApi", Scope{Game: "valheim"}, nil); got != "" {
		t.Errorf("another game = %q, want unset", got)
	}
	if got := ResolveAt(s, "graphicsApi", Scope{Game: "peak"}, map[string]string{"graphicsApi": "vulkan"}); got != "vulkan" {
		t.Errorf("profile override = %q", got)
	}
}
