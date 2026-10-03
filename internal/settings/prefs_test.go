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
