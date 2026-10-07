package launchsvc

import (
	"encoding/json"
	"testing"
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
