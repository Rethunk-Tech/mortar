package settings

import "testing"

func TestResolveOrder(t *testing.T) {
	t.Parallel()
	var empty Settings
	if got := Resolve(empty, "defaultLaunchMethod", GameStardew, nil); got != LaunchSteam {
		t.Fatalf("default = %q", got)
	}
	game := Settings{}
	putGame(&game, GameStardew, GameSettings{DefaultLaunchMethod: LaunchDirect})
	if got := Resolve(game, "defaultLaunchMethod", GameStardew, nil); got != LaunchDirect {
		t.Fatalf("game = %q", got)
	}
	if got := Resolve(game, "defaultLaunchMethod", GameStardew, map[string]string{"defaultLaunchMethod": LaunchSteam}); got != LaunchSteam {
		t.Fatalf("profile = %q", got)
	}
}

func TestResolveSkipPlayCheck(t *testing.T) {
	t.Parallel()
	var empty Settings
	if got := Resolve(empty, "skipPlayCheck", GameStardew, nil); got != "false" {
		t.Fatalf("default skip = %q", got)
	}
	game := Settings{}
	putGame(&game, GameStardew, GameSettings{SkipPlayCheck: true})
	if got := Resolve(game, "skipPlayCheck", GameStardew, nil); got != "true" {
		t.Fatalf("game skip = %q", got)
	}
	if got := Resolve(game, "skipPlayCheck", GameStardew, map[string]string{"skipPlayCheck": "false"}); got != "false" {
		t.Fatalf("profile skip = %q", got)
	}
}

func TestProfileOverridableKeys(t *testing.T) {
	t.Parallel()
	for _, key := range []string{
		"defaultLaunchMethod", "showSmapiConsole", "backupBeforePlay",
		"launchBackupsKept", "updateModsBeforePlayDefault", "skipPlayCheck",
	} {
		if !ProfileOverridable(key) {
			t.Fatalf("%s should be profile-overridable", key)
		}
	}
	if ProfileOverridable("runsKept") {
		t.Fatal("runsKept must not be profile-overridable")
	}
}
