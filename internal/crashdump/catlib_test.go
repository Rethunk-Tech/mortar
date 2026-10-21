package crashdump

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const catlibReport = "A cozy heading\r\n\r\nTime: 2026-10-21 10:30:43\r\nExit code: 0xC0000005, access violation\r\n" +
	"Module: KERNELBASE.dll + 0x1c408a, exception 0xc0000005, seen by the crash watcher\r\n" +
	"Method: RaiseException + 0x8a\r\n" +
	"Called from: UnityPlayer.dll + 0xdb437  Utils_CUSTOM_ForceCrash + 0x17, the first frame outside Windows and the C runtime\r\n" +
	"Thread: 100, the game thread\r\nGame: 1.0, CatLib 0.9\r\n\r\nMods:\r\n  Better Repair 0.1.1\r\n"

func TestParseCatLibReportNamesTheCallerOfASystemFault(t *testing.T) {
	f, ok := ParseCatLibReport(catlibReport)
	if !ok || f.Module != "UnityPlayer.dll" || f.Detail != "exception 0xc0000005 at 0x1c408a, RaiseException + 0x8a" ||
		f.CalledFrom != "UnityPlayer.dll + 0xdb437  Utils_CUSTOM_ForceCrash + 0x17" {
		t.Errorf("fault = %+v ok=%v", f, ok)
	}
	if _, ok := ParseCatLibReport("no fault here"); ok {
		t.Error("a report without a Module line named a fault")
	}
}

func TestCatLibTakesTheNewestReportSinceTheRun(t *testing.T) {
	profile := t.TempDir()
	crashes := filepath.Join(profile, "BepInEx", "CatLib", "Crashes")
	old, fresh := filepath.Join(crashes, "2026-10-20_09-00-00_11"), filepath.Join(crashes, "2026-10-21_10-30-43_19096")
	for _, d := range []string{old, fresh} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "report.txt"), []byte(catlibReport), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	since := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, since.Add(-time.Hour), since.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	f, ok := CatLib(profile, since)
	if !ok || f.ReportDir != fresh || f.Module != "UnityPlayer.dll" {
		t.Errorf("fault = %+v ok=%v", f, ok)
	}
	if _, ok := CatLib(profile, time.Now().Add(time.Hour)); ok {
		t.Error("a report older than the run was used")
	}
}
