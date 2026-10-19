package settings

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
)

func TestRegistryDefaultsMatchToday(t *testing.T) {
	s := Defaults()
	g := s.GamePrefs("stardew")
	if s.OnPlay != OnPlayStay || s.Density != DensityComfortable || s.Theme != ThemeDark || s.Dates != DatesRelative {
		t.Fatalf("app defaults: onPlay=%q density=%q theme=%q dates=%q", s.OnPlay, s.Density, s.Theme, s.Dates)
	}
	if s.SmapiBuilds != SmapiBuildsShow || g.DefaultLaunchMethod != LaunchDirect || g.ConsoleLevel != ConsoleLevelWarn {
		t.Fatalf("game defaults: smapi=%q launch=%q level=%q", s.SmapiBuilds, g.DefaultLaunchMethod, g.ConsoleLevel)
	}
	if g.RunsKept != DefaultRunsKept || g.ConsoleLogCap != DefaultConsoleLogCap || g.SaveBackupsKept != DefaultSaveBackupsKept {
		t.Fatalf("game ints: runs=%d cap=%d backups=%d", g.RunsKept, g.ConsoleLogCap, g.SaveBackupsKept)
	}
	if g.BackupBeforePlay != BackupBeforePlayChanged || g.EnableRequirements != EnableReqAlways || g.MissingRequirements != MissingReqAsk {
		t.Fatalf("game enums: backup=%q enable=%q missing=%q", g.BackupBeforePlay, g.EnableRequirements, g.MissingRequirements)
	}
	if !ToggleOn(s.ShowSmapiConsole) || ToggleOn(g.ConsoleTimestamps) || !ToggleOn(g.ConsoleFollow) {
		t.Fatal("console toggles: SMAPI console and follow on, timestamps off")
	}
}

func TestRegistryValidation(t *testing.T) {
	s := Defaults()
	if err := ApplyKeyGame(&s, "density", "huge", ""); err == nil {
		t.Fatal("expected density reject")
	}
	if err := ApplyKeyGame(&s, "smapiBuilds", "nope", "stardew"); err == nil {
		t.Fatal("expected smapiBuilds reject")
	}
	if err := ApplyKeyGame(&s, "runsKept", "0", "stardew"); err == nil {
		t.Fatal("expected runsKept reject")
	}
	if err := ApplyKeyGame(&s, "runsKept", "5", ""); err == nil {
		t.Fatal("expected missing --game")
	}
	if err := ApplyKeyGame(&s, "smapiBuilds", SmapiBuildsNever, ""); err != nil {
		t.Fatal(err)
	}
	got, err := s.LookupGame("smapiBuilds", "")
	if err != nil || got != SmapiBuildsNever {
		t.Fatalf("lookup %q %v", got, err)
	}
}

func TestBatchPrefDefaultsAndSet(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	s := Defaults()
	g := s.GamePrefs("stardew")
	if s.ProfileOrder != ProfileOrderManual || s.AutoRetryDownloads != AutoRetryOff || g.PauseDownloadsWhilePlaying {
		t.Fatalf("order/retry/pause defaults: %q %q %v", s.ProfileOrder, s.AutoRetryDownloads, g.PauseDownloadsWhilePlaying)
	}
	if s.SidebarBadges != SidebarBadgesAll || !s.VerifyNexusMD5 || s.LaunchAtLogin || s.StartMinimised || !s.RememberWindow {
		t.Fatal("badge / verify / session defaults")
	}
	if !ToggleOn(s.ShareIncludeDisabledMods) || !ToggleOn(s.ShareIncludeFomodChoices) || !ToggleOn(s.ShareIncludeNotes) || !ToggleOn(s.ShareIncludeConfigFiles) || !ToggleOn(s.ShareIncludeProblemChoices) {
		t.Fatal("share content defaults")
	}
	if s.ExtensionConnection != ExtensionAllow || g.BackupLocation != "" || g.ConflictScanDepth != ConflictScanFull {
		t.Fatalf("extension/backup/scan defaults: %q %q %q", s.ExtensionConnection, g.BackupLocation, g.ConflictScanDepth)
	}
	if err := ApplyKeyGame(&s, "autoRetryDownloads", AutoRetry3, ""); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "pauseDownloadsWhilePlaying", "true", "stardew"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "shareIncludeDisabledMods", "true", ""); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "extensionConnection", ExtensionOff, ""); err != nil {
		t.Fatal(err)
	}
	retry, err := s.Lookup("autoRetryDownloads")
	if err != nil || retry != AutoRetry3 {
		t.Fatalf("auto-retry %q %v", retry, err)
	}
	pause, err := s.LookupGame("pauseDownloadsWhilePlaying", "stardew")
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
	if err := ApplyKeyGame(&s, "smapiBuilds", SmapiBuildsInclude, "stardew"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "density", DensityCompact, ""); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "theme", ThemeLight, ""); err != nil {
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
	if in.SmapiBuilds != SmapiBuildsInclude {
		t.Fatalf("game smapi %q", in.SmapiBuilds)
	}
}

func TestRegistryDefaultsMatchDefaults(t *testing.T) {
	d := Defaults()
	for _, p := range registry {
		if got := p.get(d, ""); p.spec.Default != got {
			t.Errorf("%s: spec default %q, Defaults() %q", p.spec.Key, p.spec.Default, got)
		}
		for game, want := range p.spec.GameDefaults {
			if got := p.get(d, game); got != want {
				t.Errorf("%s/%s: spec game default %q, Defaults() %q", p.spec.Key, game, want, got)
			}
		}
	}
}

func TestNewDefaults(t *testing.T) {
	d := Defaults()
	want := map[string]string{
		"keepInTray": "true", "rememberWindow": "true", "includeBetaReleases": "true",
		"updateCheckIntervalMinutes": "180", "askEndorseMods": "false", "notifyModUpdates": "true",
		"notifyDownloadFinished": "false", "listColumns": "on,name,version,category,installed",
		"lanSharing": "true", "shareIncludeDisabledMods": "true", "lanPort": "0", "theme": "dark", "storeRetentionDays": "30",
	}
	for k, v := range want {
		if got, err := d.Lookup(k); err != nil || got != v {
			t.Errorf("%s = %q (%v), want %q", k, got, err, v)
		}
	}
	if got, _ := d.LookupGame("defaultLaunchMethod", "stardew"); got != LaunchDirect {
		t.Errorf("stardew launch = %q", got)
	}
	if got, _ := d.LookupGame("defaultLaunchMethod", "valheim"); got != LaunchSteam {
		t.Errorf("valheim launch = %q", got)
	}
}

// Every bounded setting takes both of its ends and refuses one past either, through the key setter the CLI and the
// window use.
func TestEveryIntSettingTakesItsBoundsAndRefusesBeyond(t *testing.T) {
	for _, spec := range PrefSpecs() {
		if spec.Type != TypeInt {
			continue
		}
		game := ""
		if spec.Scope == ScopeGame {
			game = "stardew"
		}
		for n, ok := range map[int]bool{spec.Min: true, spec.Max: true, spec.Min - 1: false, spec.Max + 1: false} {
			s := Defaults()
			if err := ApplyKeyGame(&s, spec.Key, strconv.Itoa(n), game); (err == nil) != ok {
				t.Errorf("%s = %d: err %v, want accepted %v", spec.Key, n, err, ok)
			}
		}
	}
}

// A game's saved settings carry every field set on them, so no per-game choice is lost on load.
func TestMergeGameKeepsEverySetField(t *testing.T) {
	var src GameSettings
	v := reflect.ValueOf(&src).Elem()
	on := true
	for sf, f := range v.Fields() {
		k := f.Kind()
		if k == reflect.String {
			f.SetString("x")
		} else if k == reflect.Int || k == reflect.Int64 {
			f.SetInt(7)
		} else if k == reflect.Bool {
			f.SetBool(true)
		} else if k == reflect.Pointer && f.Type().Elem().Kind() == reflect.Bool {
			f.Set(reflect.ValueOf(&on))
		} else if k == reflect.Slice {
			f.Set(reflect.MakeSlice(f.Type(), 1, 1))
		} else {
			t.Fatalf("%s: give this test a value for a %s field", sf.Name, f.Type())
		}
	}
	var dst GameSettings
	mergeGame(&dst, src)
	d := reflect.ValueOf(dst)
	for sf, f := range v.Fields() {
		if !reflect.DeepEqual(d.FieldByIndex(sf.Index).Interface(), f.Interface()) {
			t.Errorf("%s was not merged", sf.Name)
		}
	}
}

func TestPauseDownloadsWhilePlayingIsPerGameWithProfileOverride(t *testing.T) {
	s := Defaults()
	sc := Scope{Game: "stardew", Profile: "p"}
	if got := ResolveAt(s, "pauseDownloadsWhilePlaying", sc, nil); got != "false" {
		t.Fatalf("default %q", got)
	}
	if err := ApplyKeyGame(&s, "pauseDownloadsWhilePlaying", "true", "stardew"); err != nil {
		t.Fatal(err)
	}
	if got := ResolveAt(s, "pauseDownloadsWhilePlaying", sc, nil); got != "true" {
		t.Fatalf("game on: %q", got)
	}
	if got := ResolveAt(s, "pauseDownloadsWhilePlaying", Scope{Game: "valheim"}, nil); got != "false" {
		t.Fatalf("another game: %q", got)
	}
	if got := ResolveAt(s, "pauseDownloadsWhilePlaying", sc, map[string]string{"pauseDownloadsWhilePlaying": "false"}); got != "false" {
		t.Fatalf("profile override: %q", got)
	}
}
