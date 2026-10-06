package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi"
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
		if measure, err := prepareStartup(smapi.Loader{}, "stardew", mods); err != nil || measure {
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
	if _, err := prepareStartup(smapi.Loader{}, "stardew", mods); err != nil {
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
	if measure, err := prepareStartup(smapi.Loader{}, "stardew", mods); err != nil || !measure {
		t.Fatalf("first launch measure %v err %v", measure, err)
	}
	if measure, _ := prepareStartup(smapi.Loader{}, "stardew", mods); measure {
		t.Fatal("second launch still measured")
	}
}

// untimed hides every optional capability of the loader it wraps.
type untimed struct{ loader.Loader }

func TestPrepareStartupLeavesALoaderWithoutTimingsAlone(t *testing.T) {
	profile := t.TempDir()
	mods := filepath.Join(profile, "mods")
	marker := filepath.Join(profile, startupDir, measureMarker)
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if measure, err := prepareStartup(untimed{bepinex5.Loader{}}, "lethal-company", mods); err != nil || measure {
		t.Fatalf("measure %v err %v", measure, err)
	}
	if _, err := os.Stat(filepath.Join(mods, smapiGroupConfig)); !os.IsNotExist(err) {
		t.Fatalf("wrote the SMAPI config: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("consumed the measure request: %v", err)
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
	got, err := readStartupReports(dir, scopeStartupIDs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "20261004T004413Z" || got[0].Mods == nil || got[1].Mods[0].EventMs["UpdateTicked"] != 25444 {
		t.Fatalf("%+v", got)
	}
}

func TestPrepareStartupArmsTheBepInExPatcherOnlyForAMeasuredLaunch(t *testing.T) {
	profile := t.TempDir()
	mods := filepath.Join(profile, "mods")
	dir := filepath.Join(profile, startupDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, measureMarker), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if measure, err := prepareStartup(bepinex5.Loader{}, "lethal-company", mods); err != nil || !measure {
		t.Fatalf("measure %v err %v", measure, err)
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, ".measure-launch")); err != nil || string(b) != "MainMenu" {
		t.Fatalf("request %q err %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(mods, smapiGroupConfig)); !os.IsNotExist(err) {
		t.Fatalf("wrote the SMAPI config: %v", err)
	}
	// A launch that never reached BepInEx leaves the request; the next, unmeasured launch takes it back.
	if measure, err := prepareStartup(bepinex5.Loader{}, "lethal-company", mods); err != nil || measure {
		t.Fatalf("measure %v err %v", measure, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".measure-launch")); !os.IsNotExist(err) {
		t.Fatalf("request left armed: %v", err)
	}
}

func TestBepInExReportRowsBecomeTheirPackages(t *testing.T) {
	dir := t.TempDir()
	report := `{"schema":1,"loader":"5.4.21.0","processStart":"2026-10-06T18:00:00.000Z",` +
		`"phases":{"bridgeEntry":900,"entryDone":3000,"gameLaunched":3800,"titleMenu":5000,"titleScreen":9000},"entryTimed":true,` +
		`"mods":[{"id":"com.a.core","name":"Core","version":"1.0.0","entryMs":300},{"id":"com.a.extra","name":"Extra","version":"1.0.0","entryMs":50},` +
		`{"id":"loose.plugin","name":"Loose","version":"0.1.0","entryMs":20}],"otherMs":7730}`
	if err := os.WriteFile(filepath.Join(dir, "20261006T180000Z.json"), []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := startupOwner{ID: "thunderstore:Author-Pack", Name: "Pack", Version: "2.0.0"}
	owners := func() map[string]startupOwner { return map[string]startupOwner{"com.a.core": pkg, "com.a.extra": pkg} }
	got, err := readStartupReports(dir, func(r *StartupReport) { pluginsToPackages(r, owners) })
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %+v", err, got)
	}
	mods := got[0].Mods
	if len(mods) != 2 || mods[0].ID != pkg.ID || mods[0].Name != "Pack" || mods[0].EntryMs != 350 || mods[1].ID != "bepinex:loose.plugin" || mods[1].EntryMs != 20 {
		t.Fatalf("%+v", mods)
	}
	if got[0].Phases.TitleScreen != 9000 || !got[0].EntryTimed {
		t.Fatalf("%+v", got[0])
	}
}
