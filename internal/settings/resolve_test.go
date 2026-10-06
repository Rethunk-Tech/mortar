package settings

import "testing"

func TestResolveOrder(t *testing.T) {
	t.Parallel()
	var empty Settings
	if got := ResolveAt(empty, "defaultLaunchMethod", Scope{Game: "stardew"}, nil); got != LaunchSteam {
		t.Fatalf("default = %q", got)
	}
	game := Settings{}
	putGame(&game, "stardew", GameSettings{DefaultLaunchMethod: LaunchDirect})
	if got := ResolveAt(game, "defaultLaunchMethod", Scope{Game: "stardew"}, nil); got != LaunchDirect {
		t.Fatalf("game = %q", got)
	}
	if got := ResolveAt(game, "defaultLaunchMethod", Scope{Game: "stardew"}, map[string]string{"defaultLaunchMethod": LaunchSteam}); got != LaunchSteam {
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
	} {
		if !ProfileOverridable(key) {
			t.Fatalf("%s should be profile-overridable", key)
		}
	}
	if ProfileOverridable("runsKept") {
		t.Fatal("runsKept must not be profile-overridable")
	}
}
