package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
)

func TestPrefDefaultsMatchToday(t *testing.T) {
	d := Defaults()
	if d.OnPlay != OnPlayStay || d.GamePrefs("stardew").BackupBeforePlay != BackupBeforePlayChanged {
		t.Fatalf("play defaults = %s %s", d.OnPlay, d.GamePrefs("stardew").BackupBeforePlay)
	}
	if d.GamePrefs("stardew").SaveBackupsKept != DefaultSaveBackupsKept || d.GamePrefs("stardew").UpdateModsBeforePlayDefault {
		t.Fatal("launch backup / update-before-play defaults")
	}
	if d.GamePrefs("stardew").RunsKept != DefaultRunsKept || d.GamePrefs("stardew").ConsoleLogCap != DefaultConsoleLogCap {
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
	if d.GamePrefs("stardew").NxmDefaultProfile != "" || d.DefaultModsView != ModsViewGrid {
		t.Fatal("nxm / view defaults")
	}
	if !ToggleOn(d.ConfirmRemovals) || d.GamePrefs("stardew").CosmeticConflicts != CosmeticCollapsed {
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
	if d.Density != DensityComfortable || d.Theme != ThemeDark || d.GridCardSize != GridCardMedium || !ToggleOn(d.ShowAuthorOnCards) {
		t.Fatal("density / card defaults")
	}
	if d.ReduceMotion != ReduceMotionSystem || d.ProfileHero != HeroFull {
		t.Fatal("motion / hero defaults")
	}
	if d.GamePrefs("stardew").EnableRequirements != EnableReqAlways || d.GamePrefs("stardew").MissingRequirements != MissingReqAsk {
		t.Fatal("requirement defaults")
	}
	if !ToggleOn(d.ReuseFomodChoices) || !d.DriftChecksOn() || d.SmapiBuilds != SmapiBuildsShow {
		t.Fatal("fomod / drift / smapi-build defaults")
	}
	if !d.AutoInstallMortar() || d.AutoTrackNexus || d.GamePrefs("stardew").DefaultLaunchMethod != LaunchSteam || !ToggleOn(d.ShowSmapiConsole) {
		t.Fatal("update / launch defaults")
	}
	if d.GamePrefs("stardew").ConsoleLevel != ConsoleLevelWarn || ToggleOn(d.GamePrefs("stardew").ConsoleTimestamps) || !ToggleOn(d.GamePrefs("stardew").ConsoleFollow) {
		t.Fatal("console defaults")
	}
	if d.LanName != "" || d.LanAutoAcceptPaired || d.DownloadFolder != "" {
		t.Fatal("lan / download-folder defaults")
	}
}

func TestAutoEnableRequirementsAlwaysOnly(t *testing.T) {
	always := Defaults()
	if !always.GamePrefs("stardew").AutoEnableRequirements() {
		t.Fatal("always")
	}
	ask := Defaults()
	g := ask.GamePrefs("stardew")
	g.EnableRequirements = EnableReqAsk
	putGame(&ask, "stardew", g)
	never := Defaults()
	g = never.GamePrefs("stardew")
	g.EnableRequirements = EnableReqNever
	putGame(&never, "stardew", g)
	if ask.GamePrefs("stardew").AutoEnableRequirements() || never.GamePrefs("stardew").AutoEnableRequirements() {
		t.Fatal("ask and never must not auto-enable")
	}
}

func TestPrefsExportImportRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	datadirtest.Use(t, t.TempDir())
	src := Defaults()
	src.Language = "en"
	src.Accent = "moss"
	src.IncludeBetaReleases = true
	src.IncludePrereleaseModVersions = true
	src.KeepInTray = true
	overrides := map[string]string{
		"onPlay": "hide", "backupBeforePlay": "always", "saveBackupsKept": "9",
		"updateModsBeforePlayDefault": "true", "runsKept": "10", "consoleLogCap": "5000",
		"parallelDownloads": "4", "updateCheckIntervalMinutes": "30", "checkModUpdatesOnStart": "false",
		"notifyModUpdates": "true", "keepDownloadArchives": "true", "storeRetentionDays": "7",
		"nxmDefaultProfile": "abc", "defaultModsView": "list", "listGroupBy": "author",
		"listSortColumn": "version", "listSortDir": "desc", "confirmRemovals": "false",
		"cosmeticConflicts": "hidden", "backgroundBadgeChecks": "false", "startScreen": "gameselect",
		"dates": "absolute", "trashRetentionDays": "10", "historyEventsKept": "50",
		"notifyDownloadFinished": "false", "notifyDownloadFailed": "false", "notifyRunCrashed": "false",
		"desktopDownloadFinished": "true", "desktopDownloadFailed": "false", "desktopRunCrashed": "false",
		"desktopModUpdates": "true",
		"density":           "compact", "theme": "light", "gridCardSize": "large", "showAuthorOnCards": "false",
		"reduceMotion": "always", "profileHero": "hidden", "enableRequirements": "never",
		"missingRequirements": "autodownload", "reuseFomodChoices": "false", "driftChecks": "false",
		"smapiBuilds": "include", "smapiPin": "4.0.0", "bepinex5Pin": "5.4.2100", "autoInstallMortarUpdates": "false", "autoTrackNexus": "true",
		"defaultLaunchMethod": "direct", "showSmapiConsole": "false", "skipPlayCheck": "true", "skipIntro": "true", "consoleLevel": "debug",
		"consoleTimestamps": "false", "consoleFollow": "false", "lanSharing": "true", "lanPort": "47641", "lanName": "Workshop",
		"lanAutoAcceptPaired": "true", "downloadFolder": "/var/tmp/mortar-dl",
		"profileOrder": "name", "autoRetryDownloads": "3", "pauseDownloadsWhilePlaying": "true",
		"sidebarBadges": "problems", "backupLocation": "/var/tmp/mortar-bak", "conflictScanDepth": "skipImages",
		"shareIncludeDisabledMods": "true", "shareIncludeFomodChoices": "false", "shareIncludeNotes": "false",
		"shareIncludeConfigFiles": "false", "shareIncludeProblemChoices": "false", "verifyNexusMD5": "true", "launchAtLogin": "true",
		"startMinimised": "true", "rememberWindow": "true", "extensionConnection": "off",
		"offerNewDownloads": "false", "updateDigest": "each", "extraModsFolder": "/var/tmp/mortar-extra", "showDotHiddenMods": "true", "oldFilesOnUpdate": "keep",
		"saveBackupHours": "6", "saveBackupKeep": "3", "sourceOrder": "github,nexus", "watchFolders": "/var/tmp/mortar-watch", "syncFolder": "/var/tmp/mortar-sync", "browseFilters": "installed=hide", "showAdultContent": "true",
	}
	vortex := filepath.Join(t.TempDir(), "Vortex")
	if err := os.MkdirAll(filepath.Join(vortex, "state.v2"), 0o750); err != nil {
		t.Fatal(err)
	}
	overrides["vortexFolder"] = vortex
	if len(overrides) != len(registry) {
		t.Fatalf("%d overrides for %d registered keys", len(overrides), len(registry))
	}
	for _, p := range registry {
		v, ok := overrides[p.spec.Key]
		if !ok {
			t.Fatalf("round-trip missing override for %s", p.spec.Key)
		}
		game := ""
		if p.spec.Scope == ScopeGame {
			game = "stardew"
		}
		if err := p.set(&src, game, v); err != nil {
			t.Fatalf("%s: %v", p.spec.Key, err)
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
	for _, key := range registry {
		game := ""
		if key.spec.Scope == ScopeGame {
			game = "stardew"
		}
		want, err := src.LookupGame(key.spec.Key, game)
		if err != nil {
			t.Fatal(err)
		}
		have, err := got.LookupGame(key.spec.Key, game)
		if err != nil || have != want {
			t.Fatalf("%s: want %q got %q %v", key.spec.Key, want, have, err)
		}
	}
}

func TestOmittedPrefsNormalizeToToday(t *testing.T) {
	s, _ := open(t)
	raw := `{"global":{"accent":"sand"}}`
	if err := writeRaw(t, s, raw); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := s2.Get()
	if got.OnPlay != OnPlayStay || got.GamePrefs("stardew").BackupBeforePlay != BackupBeforePlayChanged || got.GamePrefs("stardew").RunsKept != DefaultRunsKept {
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

func TestDesktopNotifyPrefKeys(t *testing.T) {
	var s Settings
	want := map[string]string{
		"desktopDownloadFinished": "false",
		"desktopDownloadFailed":   "true",
		"desktopRunCrashed":       "true",
		"desktopModUpdates":       "false",
	}
	for key, def := range want {
		got, err := s.Lookup(key)
		if err != nil || got != def {
			t.Fatalf("%s = %q %v, want %s", key, got, err, def)
		}
	}
}

func TestPrefKeysRoundTrip(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	if err := svc.SetByKey("onPlay", "hide", ""); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get().Lookup("onPlay")
	if err != nil || got != "hide" {
		t.Fatalf("lookup = %s %v", got, err)
	}
	if err := svc.SetByKey("onPlay", "jump", ""); err == nil {
		t.Fatal("expected reject")
	}
}

func writeRaw(t *testing.T, s *Store, raw string) error {
	t.Helper()
	return os.WriteFile(s.path, []byte(raw), 0o600)
}

func TestWatchedFoldersKeepsAbsoluteUniquePaths(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	s := Settings{WatchFolders: strings.Join([]string{a, "rel", a, " " + b + " "}, string(os.PathListSeparator))}
	if got := s.WatchedFolders(); !slices.Equal(got, []string{a, b}) {
		t.Errorf("WatchedFolders = %v", got)
	}
}

func TestVortexFolderMustHoldState(t *testing.T) {
	dir := t.TempDir()
	var s Settings
	p := vortexFolderPref()
	if err := p.set(&s, "", dir); err == nil {
		t.Fatal("folder without state.v2 accepted")
	}
	if err := os.MkdirAll(filepath.Join(dir, "state.v2"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := p.set(&s, "", dir); err != nil || s.VortexFolder != dir {
		t.Fatalf("valid folder: %v, %q", err, s.VortexFolder)
	}
	if err := p.set(&s, "", ""); err != nil || s.VortexFolder != "" {
		t.Fatalf("clear: %v, %q", err, s.VortexFolder)
	}
}
