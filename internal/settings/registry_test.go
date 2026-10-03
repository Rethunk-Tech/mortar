package settings

import "testing"

func TestRegistryDefaultsMatchToday(t *testing.T) {
	s := Defaults()
	g := s.GamePrefs(GameStardew)
	if s.OnPlay != OnPlayStay || s.Density != DensityComfortable || s.Theme != ThemeDark || s.Dates != DatesRelative {
		t.Fatalf("app defaults: onPlay=%q density=%q theme=%q dates=%q", s.OnPlay, s.Density, s.Theme, s.Dates)
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

func TestBatchPrefDefaultsAndSet(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s := Defaults()
	g := s.GamePrefs(GameStardew)
	if s.ProfileOrder != ProfileOrderManual || s.AutoRetryDownloads != AutoRetryOff || s.PauseDownloadsWhilePlaying {
		t.Fatalf("order/retry/pause defaults: %q %q %v", s.ProfileOrder, s.AutoRetryDownloads, s.PauseDownloadsWhilePlaying)
	}
	if s.SidebarBadges != SidebarBadgesAll || s.VerifyNexusMD5 || s.LaunchAtLogin || s.StartMinimised || s.RememberWindow {
		t.Fatal("badge / verify / session defaults")
	}
	if ToggleOn(s.ShareIncludeDisabledMods) || !ToggleOn(s.ShareIncludeFomodChoices) || !ToggleOn(s.ShareIncludeNotes) || !ToggleOn(s.ShareIncludeConfigFiles) {
		t.Fatal("share content defaults")
	}
	if s.ExtensionConnection != ExtensionAllow || g.BackupLocation != "" || g.ConflictScanDepth != ConflictScanFull {
		t.Fatalf("extension/backup/scan defaults: %q %q %q", s.ExtensionConnection, g.BackupLocation, g.ConflictScanDepth)
	}
	if err := ApplyKey(&s, "autoRetryDownloads", AutoRetry3); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKey(&s, "pauseDownloadsWhilePlaying", "true"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKey(&s, "shareIncludeDisabledMods", "true"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKey(&s, "extensionConnection", ExtensionOff); err != nil {
		t.Fatal(err)
	}
	retry, err := s.Lookup("autoRetryDownloads")
	if err != nil || retry != AutoRetry3 {
		t.Fatalf("auto-retry %q %v", retry, err)
	}
	pause, err := s.Lookup("pauseDownloadsWhilePlaying")
	if err != nil || pause != "true" {
		t.Fatalf("pause %q %v", pause, err)
	}
	disabled, err := s.Lookup("shareIncludeDisabledMods")
	if err != nil || disabled != "true" {
		t.Fatalf("share disabled %q %v", disabled, err)
	}
	ext, err := s.Lookup("extensionConnection")
	if err != nil || ext != ExtensionOff {
		t.Fatalf("extension %q %v", ext, err)
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
	if err := ApplyKey(&s, "theme", ThemeLight); err != nil {
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
	if in.Theme != ThemeLight {
		t.Fatalf("theme %q", in.Theme)
	}
	if in.GamePrefs(GameStardew).SmapiBuilds != SmapiBuildsInclude {
		t.Fatalf("game smapi %q", in.GamePrefs(GameStardew).SmapiBuilds)
	}
}
