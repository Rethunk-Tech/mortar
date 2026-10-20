package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
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
	if d.UpdateCheckIntervalMinutes != DefaultUpdateCheckIntervalMinutes || !ToggleOn(d.NotifyModUpdates) {
		t.Fatal("update-check defaults")
	}
	if d.KeepDownloadArchives || d.StoreRetentionDays != DefaultStoreRetentionDays {
		t.Fatal("archive / store defaults")
	}
	if d.GamePrefs("stardew").NxmDefaultProfile != "" || d.DefaultModsView != ModsViewGrid || d.ImplicitModsView != ImplicitModsNested {
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
	if ToggleOn(d.NotifyDownloadFinished) || !ToggleOn(d.NotifyDownloadFailed) || !ToggleOn(d.NotifyRunCrashed) {
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
	if !d.AutoInstallMortar() || d.AutoTrackNexus || d.GamePrefs("stardew").DefaultLaunchMethod != LaunchDirect || d.GamePrefs("valheim").DefaultLaunchMethod != LaunchSteam || !ToggleOn(d.ShowSmapiConsole) {
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
		"parallelDownloads": "4", "downloadSpeedLimit": "5", "pauseDownloadsWhilePlaying": "true", "updateCheckIntervalMinutes": "30", "checkModUpdatesOnStart": "false",
		"notifyModUpdates": "true", "keepDownloadArchives": "true", "storeRetentionDays": "7",
		"nxmDefaultProfile": "abc", "defaultModsView": "list", "implicitModsView": "nested", "listGroupBy": "author",
		"listSortColumn": "version", "listSortDir": "desc", "confirmRemovals": "false",
		"cosmeticConflicts": "hidden", "backgroundBadgeChecks": "false", "startScreen": "gameselect", "antivirus": "clamd", "antivirusSocket": "/var/tmp/clamd.sock", "antivirusCommand": "scan {path}",
		"dates": "absolute", "trashRetentionDays": "10", "historyEventsKept": "50",
		"notifyDownloadFinished": "false", "notifyDownloadFailed": "false", "notifyRunCrashed": "false",
		"desktopDownloadFinished": "true", "desktopDownloadFailed": "false", "desktopRunCrashed": "false",
		"desktopModUpdates": "true",
		"density":           "compact", "theme": "light", "gridCardSize": "large", "showAuthorOnCards": "false", "browseCardStats": "size,updated",
		"reduceMotion": "always", "growGameRows": "false", "profileHero": "hidden", "enableRequirements": "never",
		"missingRequirements": "autodownload", "reuseFomodChoices": "false", "driftChecks": "false",
		"smapiBuilds": "include", "smapiPin": "4.0.0", "bepinex5Pin": "5.4.2100", "melonloaderPin": "0.7.2", "returnofmoddingPin": "1.1.30", "gdweavePin": "2.0.14", "lovelyPin": "0.10.0", "northstarPin": "1.31.14", "rivetPin": "0.1.17", "ummPin": "1.1.0", "shimloaderPin": "1.1.7", "autoInstallMortarUpdates": "false", "autoTrackNexus": "true",
		"defaultLaunchMethod": "direct", "instances": "2", "delaySeconds": "8", "showSmapiConsole": "false", "skipPlayCheck": "true", "skipIntro": "true", "graphicsApi": "dx12", "protonVersion": "Proton 9.0", "consoleLevel": "debug",
		"consoleTimestamps": "false", "consoleFollow": "false", "lanSharing": "true", "lanPort": "47641", "lanAllowAnyAddress": "true", "lanName": "Workshop",
		"lanAutoAcceptPaired": "true", "downloadFolder": "/var/tmp/mortar-dl",
		"profileOrder": "name", "autoRetryDownloads": "3",
		"sidebarBadges": "problems", "backupLocation": "/var/tmp/mortar-bak", "conflictScanDepth": "skipImages",
		"shareIncludeDisabledMods": "true", "shareIncludeFomodChoices": "false", "shareIncludeNotes": "false",
		"shareIncludeConfigFiles": "false", "shareIncludeProblemChoices": "false", "shareIncludeLibraries": "false", "verifyNexusMD5": "true", "launchAtLogin": "true",
		"startMinimised": "true", "rememberWindow": "true", "extensionConnection": "off",
		"offerNewDownloads": "false", "updateDigest": "each", "extraModsFolder": "/var/tmp/mortar-extra", "showDotHiddenMods": "true", "oldFilesOnUpdate": "keep", "gameSettingsMode": "warn", "font": "Open Sans", "cacheClearing": "off", "dependencyVersions": "exact", "thunderstoreHost": "gcdn.thunderstore.io",
		"saveBackupHours": "6", "saveBackupKeep": "3", "sourceOrder": "github,nexus", "watchFolders": "/var/tmp/mortar-watch", "syncFolder": "/var/tmp/mortar-sync", "browseFilters": "installed=hide", "browsePicks": `{"include":["Maps"],"exclude":[],"sort":"downloads"}`, "showAdultContent": "true",
		"keepInTray": "false", "includeBetaReleases": "false", "includePrereleaseModVersions": "true", "askEndorseMods": "true", "listColumns": "on,name,version",
		"checkOnlyEnabledMods": "true", "enableModsWhenInstalled": "false", "tellWhenLoaderOut": "false",
	}
	vortex := filepath.Join(t.TempDir(), "Vortex")
	if err := os.MkdirAll(filepath.Join(vortex, "state.v2"), 0o750); err != nil {
		t.Fatal(err)
	}
	overrides["vortexFolder"] = vortex
	managers := t.TempDir()
	if err := os.WriteFile(filepath.Join(managers, "data.sqlite3"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	overrides["packFolder"] = managers
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
		// A command to run never travels in an export: an imported file must not be able to name a program.
		if key.spec.Key == "antivirusSocket" || key.spec.Key == "antivirusCommand" {
			if have, _ := got.LookupGame(key.spec.Key, ""); have != "" {
				t.Fatalf("%s travelled in the export: %q", key.spec.Key, have)
			}
			continue
		}
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

func TestPackFolderMustHoldAManagersData(t *testing.T) {
	dir := t.TempDir()
	var s Settings
	p := packFolderPref()
	if err := p.set(&s, "", dir); err == nil {
		t.Fatal("an empty folder accepted")
	}
	if err := os.MkdirAll(filepath.Join(dir, "LethalCompany", "profiles"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := p.set(&s, "", dir); err != nil || s.PackFolder != dir {
		t.Fatalf("an r2modman data folder: %v, %q", err, s.PackFolder)
	}
	if err := p.set(&s, "", ""); err != nil || s.PackFolder != "" {
		t.Fatalf("clear: %v, %q", err, s.PackFolder)
	}
}

func TestGameSettingsModeDefaultsToEditAndRejectsOtherValues(t *testing.T) {
	if got := Defaults().GamePrefs("any").GameSettingsMode; got != GameSettingsEdit {
		t.Fatalf("default = %q", got)
	}
	var s Settings
	if err := ApplyKeyGame(&s, "gameSettingsMode", "sometimes", "any"); err == nil {
		t.Fatal("an unknown mode was accepted")
	}
	if !ProfileOverridable("gameSettingsMode") {
		t.Fatal("a profile must be able to override the mode")
	}
}

func TestCacheClearingIsPerGameWithProfileOverrideAndRejectsOtherValues(t *testing.T) {
	var s Settings
	if err := ApplyKeyGame(&s, "cacheClearing", "sometimes", "any"); err == nil {
		t.Fatal("an unknown value was accepted")
	}
	if !ProfileOverridable("cacheClearing") {
		t.Fatal("a profile must be able to override it")
	}
	if got := Defaults().GamePrefs("any").CacheClearing; got != CacheClearingAuto {
		t.Fatalf("code default = %q", got)
	}
}

// Each range is inclusive at both ends: a value on the bound is kept by
// normalize and accepted by validate, never reset to its default.
func TestRangeBoundsAreInclusive(t *testing.T) {
	for _, end := range []string{"min", "max"} {
		pick := func(lo, hi int) int {
			if end == "min" {
				return lo
			}
			return hi
		}
		s := defaultPrefs()
		normalizePrefs(&s)
		s.ParallelDownloads = pick(MinParallelDownloads, MaxParallelDownloads)
		s.UpdateCheckIntervalMinutes = pick(MinUpdateCheckIntervalMinutes, MaxUpdateCheckIntervalMinutes)
		s.StoreRetentionDays = pick(MinStoreRetentionDays, MaxStoreRetentionDays)
		s.TrashRetentionDays = pick(MinTrashRetentionDays, MaxTrashRetentionDays)
		s.HistoryEventsKept = pick(MinHistoryEventsKept, MaxHistoryEventsKept)
		if err := validatePrefs(s); err != nil {
			t.Errorf("%s prefs: validate: %v", end, err)
		}
		got := s
		normalizePrefs(&got)
		if got.ParallelDownloads != s.ParallelDownloads || got.UpdateCheckIntervalMinutes != s.UpdateCheckIntervalMinutes ||
			got.StoreRetentionDays != s.StoreRetentionDays || got.TrashRetentionDays != s.TrashRetentionDays ||
			got.HistoryEventsKept != s.HistoryEventsKept {
			t.Errorf("%s prefs: normalize changed a bound value", end)
		}

		g := defaultGameSettings()
		g.SaveBackupsKept = pick(MinSaveBackupsKept, MaxSaveBackupsKept)
		g.SaveBackupKeep = pick(MinSaveBackupsKept, MaxSaveBackupsKept)
		g.SaveBackupHours = pick(MinSaveBackupHours, MaxSaveBackupHours)
		g.RunsKept = pick(MinRunsKept, MaxRunsKept)
		g.ConsoleLogCap = pick(MinConsoleLogCap, MaxConsoleLogCap)
		g.Instances = pick(MinInstances, MaxInstances)
		g.DelaySeconds = pick(MinDelaySeconds, MaxDelaySeconds)
		if err := validateGame(g); err != nil {
			t.Errorf("%s game: validate: %v", end, err)
		}
		gotG := g
		normalizeGame("", &gotG)
		if gotG.SaveBackupsKept != g.SaveBackupsKept || gotG.SaveBackupKeep != g.SaveBackupKeep ||
			gotG.SaveBackupHours != g.SaveBackupHours || gotG.RunsKept != g.RunsKept ||
			gotG.ConsoleLogCap != g.ConsoleLogCap || gotG.Instances != g.Instances || gotG.DelaySeconds != g.DelaySeconds {
			t.Errorf("%s game: normalize changed a bound value", end)
		}
	}
}

func TestHidingAModFromBrowseIsPerGameAndUndoable(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	m := HiddenMod{Source: "thunderstore", ID: "Me-Mod", Name: "Mod"}
	for range 2 {
		if err := svc.HideBrowseMod("a", m); err != nil {
			t.Fatal(err)
		}
	}
	got := st.Get()
	if len(got.GamePrefs("a").HiddenMods) != 1 || !got.GamePrefs("a").HidesMod("Thunderstore", "me-mod") || got.GamePrefs("b").HidesMod(m.Source, m.ID) {
		t.Fatalf("hidden = %+v", got.Games)
	}
	if err := svc.UnhideBrowseMod("a", "thunderstore", "ME-MOD"); err != nil {
		t.Fatal(err)
	}
	if st.Get().GamePrefs("a").HidesMod(m.Source, m.ID) {
		t.Fatal("still hidden")
	}
}

func TestHidingAnAuthorMatchesWithoutCaseAndUndoes(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	for _, a := range []string{"Me", "me"} {
		if err := svc.HideBrowseAuthor("a", a); err != nil {
			t.Fatal(err)
		}
	}
	gp := st.Get().GamePrefs("a")
	if len(gp.HiddenAuthors) != 1 || !gp.HidesAuthor("ME") || gp.HidesAuthor("") || st.Get().GamePrefs("b").HidesAuthor("me") {
		t.Fatalf("hidden authors = %v", gp.HiddenAuthors)
	}
	if err := svc.UnhideBrowseAuthor("a", "ME"); err != nil {
		t.Fatal(err)
	}
	if st.Get().GamePrefs("a").HidesAuthor("me") {
		t.Fatal("still hidden")
	}
}

func TestHidingAGameIsUndoable(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	for _, hide := range []bool{true, true, false} {
		if err := svc.SetGameHidden("a", hide); err != nil {
			t.Fatal(err)
		}
		if err := svc.SetGameHidden("b", true); err != nil {
			t.Fatal(err)
		}
	}
	if got := st.Get().HiddenGames; len(got) != 1 || got[0] != "b" {
		t.Fatalf("hidden games = %v", got)
	}
}

func TestThunderstoreHostChoicesAreTheDownloadCDNs(t *testing.T) {
	var want []string
	for _, base := range thunderstore.CDNs {
		want = append(want, strings.TrimPrefix(base, "https://"))
	}
	if got := thunderstoreHostValues[1:]; !slices.Equal(got, want) {
		t.Fatalf("host choices %v, CDNs %v", got, want)
	}
}

func TestStarringAModIsPerGameAndUndoable(t *testing.T) {
	st, _ := open(t)
	svc := NewService(st)
	key := FavouriteKey("Thunderstore", "Me-Mod")
	for range 2 {
		if err := svc.SetModFavourite("a", key, true); err != nil {
			t.Fatal(err)
		}
	}
	if got := st.Get().GamePrefs("a").FavouriteMods; len(got) != 1 || got[0] != "thunderstore:me-mod" {
		t.Fatalf("favourites = %v", got)
	}
	if len(st.Get().GamePrefs("b").FavouriteMods) != 0 {
		t.Fatal("another game got the star")
	}
	if err := svc.SetModFavourite("a", "THUNDERSTORE:ME-MOD", false); err != nil {
		t.Fatal(err)
	}
	if len(st.Get().GamePrefs("a").FavouriteMods) != 0 {
		t.Fatal("still starred")
	}
}
