//go:build !windows

package launchsvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/bridge/bridgetest"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestPerfRowsNameAndAddUpEachPackagesPlugins(t *testing.T) {
	var got bridgePerf
	fixture := `{"measured":true,"seconds":60,"frames":3600,"fps":60,"frameMs":{"avg":16.7,"p50":16.6,"p95":20,"p99":33,"max":80},` +
		`"monoUsedBytes":104857600,"monoHeapBytes":209715200,"gcCollections":4,` +
		`"baseline":{"frameMs":16.7,"modsMs":2.25,"withoutModsMs":14.45},"plugins":[` +
		`{"guid":"com.a.core","msPerFrame":0.5,"p95Ms":1,"peakMs":2,"share":0.03,"callsPerFrame":10,"patches":3,"transpilers":0},` +
		`{"guid":"com.a.extra","msPerFrame":0.25,"p95Ms":0.5,"peakMs":1,"share":0.015,"callsPerFrame":2,"patches":1,"transpilers":1},` +
		`{"guid":"loose.plugin","msPerFrame":1.5,"p95Ms":2,"peakMs":4,"share":0.09,"callsPerFrame":1,"patches":0,"transpilers":0},` +
		`{"guid":"idle.plugin","msPerFrame":0,"peakMs":0,"callsPerFrame":0,"patches":1,"transpilers":0}],` +
		`"patchOwners":{"com.a.core":["Game::Update"]}}`
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatal(err)
	}
	owners := func() map[string]startupOwner {
		pkg := startupOwner{ID: "thunderstore:Author-Pack", Name: "Pack"}
		return map[string]startupOwner{"com.a.core": pkg, "com.a.extra": pkg}
	}
	rows := perfRows(got, owners)
	if len(rows) != 2 || rows[0].Name != "loose.plugin" || rows[1].Name != "Pack" || rows[1].AverageMs != 0.75 || rows[1].PeakMs != 3 || rows[1].Calls != 12 || rows[1].P95Ms != 1.5 || rows[1].Share != 0.045 || rows[0].P95Ms != 2 {
		t.Fatalf("%+v", rows)
	}
	if got.Baseline.ModsMs != 2.25 || got.Baseline.WithoutModsMs != 14.45 {
		t.Fatalf("baseline %+v", got.Baseline)
	}
	if got.FrameMs.P99 != 33 || got.MonoHeapBytes != 209715200 {
		t.Fatalf("%+v", got)
	}
}

// Two packages can share a display name; their plugins must still add up as two rows, as the startup report's own
// package rows do.
func TestPerfRowsKeepTwoPackagesWithTheSameNameApart(t *testing.T) {
	var got bridgePerf
	fixture := `{"measured":true,"plugins":[` +
		`{"guid":"com.one.plugin","msPerFrame":1,"callsPerFrame":1},` +
		`{"guid":"com.two.plugin","msPerFrame":2,"callsPerFrame":2}]}`
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatal(err)
	}
	owners := func() map[string]startupOwner {
		return map[string]startupOwner{
			"com.one.plugin": {ID: "thunderstore:Author-One", Name: "Same"},
			"com.two.plugin": {ID: "thunderstore:Author-Two", Name: "Same"},
		}
	}
	rows := perfRows(got, owners)
	if len(rows) != 2 || rows[0].AverageMs != 2 || rows[1].AverageMs != 1 {
		t.Fatalf("%+v", rows)
	}
}

func TestPerfReplySaysWhetherTheCompanionCanStop(t *testing.T) {
	for raw, want := range map[string]bool{
		`{"stoppable":true,"measured":true}`: true,
		`{"measured":true,"frames":3}`:       false,
	} {
		var got bridgePerf
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatal(err)
		}
		if got.Stoppable != want {
			t.Fatalf("%s: Stoppable = %v", raw, got.Stoppable)
		}
	}
}

func TestSmapiOwnersReadTheInstalledListOncePerPeek(t *testing.T) {
	reads := 0
	owners := smapiOwners(func() ([]profile.Installed, error) {
		reads++
		return []profile.Installed{{Name: "Pack", Enabled: true}, {Name: "Off"}}, nil
	})
	for range 5 {
		owners()
	}
	if reads != 1 {
		t.Fatalf("the installed list was read %d times for one peek, want 1", reads)
	}
}

func TestMeasureInGameRefusesALoaderOrCompanionThatCannotMeasure(t *testing.T) {
	svc, p := startEnv(t)
	if _, err := svc.MeasureInGame("no-such-game", p.ID, false); usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("unknown game: %v", err)
	}
	if _, err := svc.PeekInGame("stardew", p.ID); err == nil {
		t.Fatal("SMAPI without a companion must fail")
	}
	g := game.Find("stardew")
	svc.mu.Lock()
	svc.logs[keyOf(g)] = session{profile: p.ID, features: bridge.Features{bridge.FeatureConsole}}
	svc.mu.Unlock()
	if _, err := svc.MeasureInGame("stardew", p.ID, true); usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("companion without perf: %v", err)
	}
}

func TestPeekAndMeasureInGameReadTheCompanion(t *testing.T) {
	set, ps := testenv.Env(t)
	svc := NewService(t.TempDir(), set, ps)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}

	idle := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok {"measured":false,"reason":"not launched","stoppable":true}`)
	live, err := svc.PeekInGame(lc, p.ID)
	if err != nil || live.Measured || live.Reason != "not launched" || !live.Stoppable || live.Rows == nil {
		t.Fatalf("unmeasured: %+v %v", live, err)
	}
	if cmd := <-idle; cmd != "perf" {
		t.Fatalf("peek sent %q", cmd)
	}

	start := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok {"measured":true,"stoppable":true}`)
	got, err := svc.MeasureInGame(lc, p.ID, true)
	if err != nil || !got.Measured || !got.Stoppable || got.Report != nil {
		t.Fatalf("start: %+v %v", got, err)
	}
	if cmd := <-start; cmd != "perf start" {
		t.Fatalf("start sent %q", cmd)
	}

	empty := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok {"measured":true,"plugins":[]}`)
	if _, err := svc.MeasureInGame(lc, p.ID, false); err == nil {
		t.Fatal("a measured game with no plugin times must say to start first")
	}
	<-empty

	summary := `ok {"measured":true,"stoppable":true,"seconds":2,"frames":120,"fps":60,` +
		`"frameMs":{"avg":16,"p50":16,"p95":20,"p99":30,"max":40},` +
		`"monoUsedBytes":1,"monoHeapBytes":2,"gcCollections":3,` +
		`"baseline":{"modsMs":1.5,"withoutModsMs":14.5},` +
		`"plugins":[{"guid":"loose.plugin","msPerFrame":1.5,"p95Ms":2,"peakMs":4,"share":0.1,"callsPerFrame":1}]}`
	ready := bridgetest.FakeBridge(t, bridge.BepInEx, dir, summary)
	peek, err := svc.PeekInGame(lc, p.ID)
	if err != nil || !peek.Measured || peek.Frame == nil || peek.Frame.FPS != 60 || len(peek.Rows) != 1 || peek.Rows[0].Name != "loose.plugin" {
		t.Fatalf("peek live: %+v %v", peek, err)
	}
	<-ready

	save := bridgetest.FakeBridge(t, bridge.BepInEx, dir, summary)
	saved, err := svc.MeasureInGame(lc, p.ID, false)
	if err != nil || saved.Report == nil || saved.Report.Frame == nil || saved.Report.Frame.ModsMs != 1.5 || len(saved.Report.Rows) != 1 {
		t.Fatalf("save: %+v %v", saved, err)
	}
	<-save

	bad := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok []`)
	if _, err := svc.PeekInGame(lc, p.ID); err == nil {
		t.Fatal("a non-object perf reply must fail")
	}
	<-bad
}

func TestStopInGameNeedsALoaderWhoseCompanionCanStop(t *testing.T) {
	set, ps := testenv.Env(t)
	svc := NewService(t.TempDir(), set, ps)
	sv := testenv.Profile(t, ps, "stardew", "A")
	if _, err := svc.StopInGame("stardew", sv.ID); err == nil {
		t.Fatal("a profile with no bridge mod stopped a measurement")
	}

	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	asked := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok {"measured":true}`)
	measured, err := svc.StopInGame(lc, p.ID)
	if err != nil || !measured {
		t.Fatalf("StopInGame = %v, %v", measured, err)
	}
	if cmd := <-asked; cmd != "perf stop" {
		t.Fatalf("the game was sent %q", cmd)
	}

	bad := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `ok []`)
	if _, err := svc.StopInGame(lc, p.ID); err == nil {
		t.Fatal("a non-object stop reply must fail")
	}
	<-bad
	nope := bridgetest.FakeBridge(t, bridge.BepInEx, dir, `error: nope`)
	if _, err := svc.StopInGame(lc, p.ID); err == nil {
		t.Fatal("a rejected stop must fail")
	}
	<-nope
	if _, err := svc.StopInGame(lc, "missing"); err == nil {
		t.Fatal("a missing profile must fail")
	}
}

func TestPeekInGameNamesSMAPIModsByInstalledID(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	svc := NewService(t.TempDir(), nil, profiles)
	src := filepath.Join(t.TempDir(), "MortarSmapiBridge")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"Name":"Bridge","Author":"m","Version":"0.9.0","UniqueID":"Rethunk.MortarSmapiBridge","EntryDll":"B.dll"}`
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir(t.Context(), "stardew", "bridge-0.9.0", filepath.Dir(src)); err != nil {
		t.Fatal(err)
	}
	if err := profiles.ApplyBundled("stardew", profile.Bundle{Key: "bridge-0.9.0", Source: profile.Source{Kind: profile.SourceMortar, Name: "Bridge"}}); err != nil {
		t.Fatal(err)
	}
	modsDir, err := profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(modsDir, "bridge-0.9.0", "MortarSmapiBridge")
	asked := bridgetest.FakeBridge(t, bridge.SMAPI, dir, `ok {"measured":true,"plugins":[{"guid":"pack","msPerFrame":1,"callsPerFrame":1}]}`)
	live, err := svc.PeekInGame("stardew", p.ID)
	if err != nil || !live.Measured || len(live.Rows) != 1 || live.Rows[0].Name != "pack" {
		t.Fatalf("peek: %+v %v", live, err)
	}
	if cmd := <-asked; cmd != "perf" {
		t.Fatalf("the game was sent %q", cmd)
	}
}
