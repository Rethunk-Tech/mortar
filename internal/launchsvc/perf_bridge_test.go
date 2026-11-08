package launchsvc

import (
	"encoding/json"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/bridge/bridgetest"
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

func TestStopInGameNeedsALoaderWhoseCompanionCanStop(t *testing.T) {
	set, ps := testenv.Env(t)
	svc := NewService(t.TempDir(), set, ps)
	sv := testenv.Profile(t, ps, "stardew", "A")
	if _, err := svc.StopInGame("stardew", sv.ID); usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("SMAPI has no stop verb, got %v", err)
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
}
