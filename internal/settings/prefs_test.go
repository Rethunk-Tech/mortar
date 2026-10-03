package settings

import (
	"os"
	"testing"
	"time"
)

func TestPrefDefaultsMatchToday(t *testing.T) {
	d := Defaults()
	if d.OnPlay != OnPlayStay || d.BackupBeforePlay != BackupBeforePlayChanged {
		t.Fatalf("play defaults = %s %s", d.OnPlay, d.BackupBeforePlay)
	}
	if d.LaunchBackupsKept != DefaultLaunchBackupsKept || d.UpdateModsBeforePlayDefault {
		t.Fatal("launch backup / update-before-play defaults")
	}
	if d.RunsKept != DefaultRunsKept || d.ConsoleLogCap != DefaultConsoleLogCap {
		t.Fatal("log defaults")
	}
	if d.ParallelDownloads != DefaultParallelDownloads {
		t.Fatalf("parallel = %d", d.ParallelDownloads)
	}
	if d.UpdateCheckIntervalMinutes != DefaultUpdateCheckIntervalMinutes || ToggleOn(d.NotifyModUpdates) {
		t.Fatal("update-check defaults")
	}
	if d.KeepDownloadArchives || d.StoreRetentionDays != DefaultStoreRetentionDays {
		t.Fatal("archive / store defaults")
	}
	if d.NxmDefaultProfile != "" || d.DefaultModsView != ModsViewGrid {
		t.Fatal("nxm / view defaults")
	}
	if !ToggleOn(d.ConfirmRemovals) || d.CosmeticConflicts != CosmeticCollapsed {
		t.Fatal("confirm / cosmetic defaults")
	}
	if !ToggleOn(d.BackgroundBadgeChecks) || d.StartScreen != StartScreenLast || d.Dates != DatesRelative {
		t.Fatal("badge / start / date defaults")
	}
	if d.TrashRetentionDays != DefaultTrashRetentionDays || d.HistoryEventsKept != DefaultHistoryEventsKept {
		t.Fatal("trash / history defaults")
	}
	if !ToggleOn(d.NotifyDownloadFinished) || !ToggleOn(d.NotifyDownloadFailed) || !ToggleOn(d.NotifyRunCrashed) {
		t.Fatal("notify defaults")
	}
	if d.Density != DensityComfortable || d.GridCardSize != GridCardMedium || !ToggleOn(d.ShowAuthorOnCards) {
		t.Fatal("density / card defaults")
	}
	if d.ReduceMotion != ReduceMotionSystem || d.ProfileHero != HeroFull {
		t.Fatal("motion / hero defaults")
	}
	if d.EnableRequirements != EnableReqAlways || d.MissingRequirements != MissingReqAsk {
		t.Fatal("requirement defaults")
	}
	if !ToggleOn(d.ReuseFomodChoices) || !d.DriftChecksOn() || d.SmapiBuilds != SmapiBuildsShow {
		t.Fatal("fomod / drift / smapi-build defaults")
	}
	if !d.AutoInstallMortar() || d.AutoTrackNexus || d.DefaultLaunchMethod != LaunchSteam || !d.ShowConsoleWindow() {
		t.Fatal("update / launch defaults")
	}
	if d.ConsoleLevel != ConsoleLevelInfo || !ToggleOn(d.ConsoleTimestamps) || !ToggleOn(d.ConsoleFollow) {
		t.Fatal("console defaults")
	}
	if d.LanName != "" || d.LanAutoAcceptSameAccount || d.DownloadFolder != "" {
		t.Fatal("lan / download-folder defaults")
	}
}

func TestAutoEnableRequirementsAlwaysOnly(t *testing.T) {
	always := Defaults()
	if !always.AutoEnableRequirements() {
		t.Fatal("always")
	}
	ask := Defaults()
	ask.EnableRequirements = EnableReqAsk
	never := Defaults()
	never.EnableRequirements = EnableReqNever
	if ask.AutoEnableRequirements() || never.AutoEnableRequirements() {
		t.Fatal("ask and never must not auto-enable")
	}
}

func TestPrefsExportImportRoundTrip(t *testing.T) {
	src := Defaults()
	src.Language = "en"
	src.Accent = "moss"
	src.IncludeBetaReleases = true
	src.IncludePrereleaseModVersions = true
	src.KeepInTray = true
	overrides := map[string]string{
		"onPlay": "hide", "backupBeforePlay": "always", "launchBackupsKept": "9",
		"updateModsBeforePlayDefault": "true", "runsKept": "10", "consoleLogCap": "5000",
		"parallelDownloads": "4", "updateCheckIntervalMinutes": "30", "checkModUpdatesOnStart": "false",
		"notifyModUpdates": "true", "keepDownloadArchives": "true", "storeRetentionDays": "7",
		"nxmDefaultProfile": "abc", "defaultModsView": "list", "listGroupBy": "author",
		"listSortColumn": "version", "listSortDir": "desc", "confirmRemovals": "false",
		"cosmeticConflicts": "hidden", "backgroundBadgeChecks": "false", "startScreen": "gameselect",
		"dates": "absolute", "trashRetentionDays": "10", "historyEventsKept": "50",
		"notifyDownloadFinished": "false", "notifyDownloadFailed": "false", "notifyRunCrashed": "false",
		"density": "compact", "gridCardSize": "large", "showAuthorOnCards": "false",
		"reduceMotion": "always", "profileHero": "hidden", "enableRequirements": "never",
		"missingRequirements": "autodownload", "reuseFomodChoices": "false", "driftChecks": "false",
		"smapiBuilds": "include", "autoInstallMortarUpdates": "false", "autoTrackNexus": "true",
		"defaultLaunchMethod": "direct", "showSmapiConsole": "false", "consoleLevel": "debug",
		"consoleTimestamps": "false", "consoleFollow": "false", "lanName": "Workshop",
		"lanAutoAcceptSameAccount": "true", "downloadFolder": "/var/tmp/mortar-dl",
	}
	for _, p := range PrefKeys() {
		v, ok := overrides[p.Key]
		if !ok {
			t.Fatalf("round-trip missing override for %s", p.Key)
		}
		if err := p.Apply(&src, v); err != nil {
			t.Fatalf("%s: %v", p.Key, err)
		}
	}
	b, err := MarshalExport(src)
	if err != nil {
		t.Fatal(err)
	}
	p, present, err := ParseExport(b)
	if err != nil {
		t.Fatal(err)
	}
	got := Defaults()
	ApplyExport(&got, p, present)
	if got.Language != src.Language || got.Accent != src.Accent || got.IncludeBetaReleases != src.IncludeBetaReleases || got.KeepInTray != src.KeepInTray {
		t.Fatal("portable-only fields did not round-trip")
	}
	for _, key := range PrefKeys() {
		want, err := src.Lookup(key.Key)
		if err != nil {
			t.Fatal(err)
		}
		have, err := got.Lookup(key.Key)
		if err != nil || have != want {
			t.Fatalf("%s: want %q got %q %v", key.Key, want, have, err)
		}
	}
}

func TestOmittedPrefsNormalizeToToday(t *testing.T) {
	s, _ := open(t)
	raw := `{"accent":"sand"}`
	if err := writeRaw(t, s, raw); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := s2.Get()
	if got.OnPlay != OnPlayStay || got.BackupBeforePlay != BackupBeforePlayChanged || got.RunsKept != DefaultRunsKept {
		t.Fatalf("normalized prefs = %+v", got)
	}
}

func TestShouldBackupBeforePlayModes(t *testing.T) {
	if !ShouldBackupBeforePlay(BackupBeforePlayAlways, false, false) {
		t.Fatal("always")
	}
	if ShouldBackupBeforePlay(BackupBeforePlayNever, true, true) {
		t.Fatal("never")
	}
	if !ShouldBackupBeforePlay(BackupBeforePlayChanged, true, false) || ShouldBackupBeforePlay(BackupBeforePlayChanged, false, false) {
		t.Fatal("changed")
	}
}

func TestUpdateCheckEveryDefaultHour(t *testing.T) {
	if Defaults().UpdateCheckEvery() != time.Hour {
		t.Fatalf("interval = %s", Defaults().UpdateCheckEvery())
	}
}

func TestStoreUnusedForZeroIsForever(t *testing.T) {
	s := Defaults()
	s.StoreRetentionDays = 0
	if s.StoreUnusedFor() != 0 {
		t.Fatal("0 days must mean forever")
	}
	s.StoreRetentionDays = 30
	if s.StoreUnusedFor() != 30*24*time.Hour {
		t.Fatal("30 days")
	}
}

func TestTrashKeepForDefaultThirtyDays(t *testing.T) {
	if Defaults().TrashKeepFor() != 30*24*time.Hour {
		t.Fatalf("trash = %s", Defaults().TrashKeepFor())
	}
}

func TestPrefKeysRoundTrip(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	if err := svc.SetByKey("onPlay", "hide"); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get().Lookup("onPlay")
	if err != nil || got != "hide" {
		t.Fatalf("lookup = %s %v", got, err)
	}
	if err := svc.SetByKey("onPlay", "jump"); err == nil {
		t.Fatal("expected reject")
	}
}

func writeRaw(t *testing.T, s *Store, raw string) error {
	t.Helper()
	return os.WriteFile(s.path, []byte(raw), 0o600)
}
