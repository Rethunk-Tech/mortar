package settings

import (
	"encoding/json"
	"testing"
)

func TestRegistryDefaultsMatchToday(t *testing.T) {
	s := Defaults()
	g := s.GamePrefs(GameStardew)
	if s.OnPlay != OnPlayStay || s.Density != DensityComfortable || s.Dates != DatesRelative {
		t.Fatalf("app defaults: onPlay=%q density=%q dates=%q", s.OnPlay, s.Density, s.Dates)
	}
	if g.SmapiBuilds != SmapiBuildsShow || g.DefaultLaunchMethod != LaunchSteam || g.ConsoleLevel != ConsoleLevelInfo {
		t.Fatalf("game defaults: smapi=%q launch=%q level=%q", g.SmapiBuilds, g.DefaultLaunchMethod, g.ConsoleLevel)
	}
	if g.RunsKept != DefaultRunsKept || g.ConsoleLogCap != DefaultConsoleLogCap || g.LaunchBackupsKept != DefaultLaunchBackupsKept {
		t.Fatalf("game ints: runs=%d cap=%d backups=%d", g.RunsKept, g.ConsoleLogCap, g.LaunchBackupsKept)
	}
	if g.BackupBeforePlay != BackupBeforePlayChanged || g.EnableRequirements != EnableReqAlways || g.MissingRequirements != MissingReqAsk {
		t.Fatalf("game enums: backup=%q enable=%q missing=%q", g.BackupBeforePlay, g.EnableRequirements, g.MissingRequirements)
	}
	if !ToggleOn(g.ShowSmapiConsole) || !ToggleOn(g.ConsoleTimestamps) || !ToggleOn(g.ConsoleFollow) {
		t.Fatal("console toggles should default on")
	}
}

func TestRegistryValidation(t *testing.T) {
	s := Defaults()
	if err := ApplyKey(&s, "density", "huge"); err == nil {
		t.Fatal("expected density reject")
	}
	if err := ApplyKeyGame(&s, "smapiBuilds", "nope", GameStardew); err == nil {
		t.Fatal("expected smapiBuilds reject")
	}
	if err := ApplyKeyGame(&s, "runsKept", "0", GameStardew); err == nil {
		t.Fatal("expected runsKept reject")
	}
	if err := ApplyKeyGame(&s, "smapiBuilds", SmapiBuildsNever, ""); err == nil {
		t.Fatal("expected missing --game")
	}
	if err := ApplyKeyGame(&s, "smapiBuilds", SmapiBuildsNever, GameStardew); err != nil {
		t.Fatal(err)
	}
	got, err := s.LookupGame("smapiBuilds", GameStardew)
	if err != nil || got != SmapiBuildsNever {
		t.Fatalf("lookup %q %v", got, err)
	}
}

func TestLegacyRootFieldsMigrateToStardew(t *testing.T) {
	raw := []byte(`{
		"smapiBuilds":"include",
		"defaultLaunchMethod":"direct",
		"runsKept":7,
		"consoleLogCap":5000,
		"backupBeforePlay":"always",
		"launchBackupsKept":3,
		"updateModsBeforePlayDefault":true,
		"nxmDefaultProfile":"p1",
		"cosmeticConflicts":"hidden",
		"enableRequirements":"ask",
		"missingRequirements":"never",
		"showSmapiConsole":false,
		"consoleLevel":"debug",
		"consoleTimestamps":false,
		"consoleFollow":false
	}`)
	var cur Settings
	if err := json.Unmarshal(raw, &cur); err != nil {
		t.Fatal(err)
	}
	migrateLegacyGameFields(&cur)
	normalizePrefs(&cur)
	if cur.LegacySmapiBuilds != "" || cur.LegacyRunsKept != 0 {
		t.Fatalf("legacy fields should clear after migrate: %+v", cur)
	}
	g := cur.GamePrefs(GameStardew)
	if g.SmapiBuilds != SmapiBuildsInclude || g.DefaultLaunchMethod != LaunchDirect || g.RunsKept != 7 {
		t.Fatalf("migrated game: %+v", g)
	}
	if g.ConsoleLogCap != 5000 || g.BackupBeforePlay != BackupBeforePlayAlways || g.LaunchBackupsKept != 3 {
		t.Fatalf("migrated launch/console: %+v", g)
	}
	if !g.UpdateModsBeforePlayDefault || g.NxmDefaultProfile != "p1" || g.CosmeticConflicts != CosmeticHidden {
		t.Fatalf("migrated more: %+v", g)
	}
	if g.EnableRequirements != EnableReqAsk || g.MissingRequirements != MissingReqNever {
		t.Fatalf("migrated reqs: %+v", g)
	}
	if ToggleOn(g.ShowSmapiConsole) || g.ConsoleLevel != ConsoleLevelDebug || ToggleOn(g.ConsoleTimestamps) || ToggleOn(g.ConsoleFollow) {
		t.Fatalf("migrated console: %+v", g)
	}
}

func TestPortableRoundTripGameScope(t *testing.T) {
	s := Defaults()
	if err := ApplyKeyGame(&s, "smapiBuilds", SmapiBuildsInclude, GameStardew); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKey(&s, "density", DensityCompact); err != nil {
		t.Fatal(err)
	}
	blob, err := MarshalExport(s)
	if err != nil {
		t.Fatal(err)
	}
	p, present, err := ParseExport(blob)
	if err != nil {
		t.Fatal(err)
	}
	in := Defaults()
	ApplyExport(&in, p, present)
	if in.Density != DensityCompact {
		t.Fatalf("density %q", in.Density)
	}
	if in.GamePrefs(GameStardew).SmapiBuilds != SmapiBuildsInclude {
		t.Fatalf("game smapi %q", in.GamePrefs(GameStardew).SmapiBuilds)
	}
}
