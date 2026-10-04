package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestPrepareStartupLoadsTheBridgeEarlyAndKeepsUserKeys(t *testing.T) {
	mods := filepath.Join(t.TempDir(), "mods")
	if err := os.MkdirAll(mods, 0o700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(mods, smapiGroupConfig)
	if err := os.WriteFile(cfgPath, []byte(`{"ModsToLoadEarly":["Some.Other"],"VerboseLogging":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if measure, err := prepareStartup(mods); err != nil || measure {
			t.Fatalf("measure %v err %v", measure, err)
		}
	}
	b, err := fsx.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Count(got, "Rethunk.MortarSmapiBridge") != 1 || !strings.Contains(got, "Some.Other") || !strings.Contains(got, "VerboseLogging") {
		t.Fatalf("config %s", got)
	}
}

func TestPrepareStartupLeavesAFileItCannotParse(t *testing.T) {
	mods := t.TempDir()
	cfgPath := filepath.Join(mods, smapiGroupConfig)
	raw := "{ // a comment SMAPI allows\n \"ModsToLoadEarly\": [] }"
	if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareStartup(mods); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(cfgPath); string(b) != raw {
		t.Fatalf("rewrote %s", b)
	}
}

func TestPrepareStartupConsumesTheMeasureRequest(t *testing.T) {
	profile := t.TempDir()
	mods := filepath.Join(profile, "mods")
	if err := os.MkdirAll(filepath.Join(profile, startupDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(mods, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, startupDir, measureMarker), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if measure, err := prepareStartup(mods); err != nil || !measure {
		t.Fatalf("first launch measure %v err %v", measure, err)
	}
	if measure, _ := prepareStartup(mods); measure {
		t.Fatal("second launch still measured")
	}
}

func TestReadStartupReportsNewestFirstSkippingBroken(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("20261004T004105Z.json", `{"schema":1,"phases":{"titleScreen":57108},"mods":[{"id":"Pathoschild.ContentPatcher","eventMs":{"UpdateTicked":25444}}]}`)
	write("20261004T004413Z.json", `{"schema":1,"phases":{"titleScreen":57320}}`)
	write("20261004T005000Z.json", `{"schema":1,`)
	got, err := readStartupReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "20261004T004413Z" || got[0].Mods == nil || got[1].Mods[0].EventMs["UpdateTicked"] != 25444 {
		t.Fatalf("%+v", got)
	}
}
