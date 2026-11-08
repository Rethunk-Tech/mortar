package launchsvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

func TestUsualReadyIsTheMedianOfTheLastFiveReadyRuns(t *testing.T) {
	runs := []Run{{ReadyMs: 40_000, Loads: 43}, {}, {ReadyMs: 10_000}, {ReadyMs: 30_000}, {ReadyMs: 20_000}, {ReadyMs: 50_000}, {ReadyMs: 900_000}}
	if ms, loads := usualReady(runs); ms != 30_000 || loads != 43 {
		t.Fatalf("usual %d ms, %d loads", ms, loads)
	}
	if ms, _ := usualReady(nil); ms != 0 {
		t.Fatalf("no runs: %d", ms)
	}
}

func TestSlowAtFollowsTheUsualTime(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	for usual, want := range map[int64]time.Duration{0: launch.SlowAfter, 5_000: 30 * time.Second, 40_000: 80 * time.Second} {
		if got := slowAt(start, usual).Sub(start); got != want {
			t.Errorf("usual %d ms: slow after %v, want %v", usual, got, want)
		}
	}
}

func TestLoadLineCountsModsUntilTheLoaderIsReady(t *testing.T) {
	var p Progress
	for _, l := range []string{
		"[Info   :   BepInEx] 2 plugins to load",
		"[Info   :   BepInEx] Loading [LethalLib 1.0.0]",
		"[Info   : LethalLib] hello",
		"[Info   :   BepInEx] Loading [MoreCompany 1.9.0]",
		"[Message:   BepInEx] Chainloader startup complete",
	} {
		loadLine(bepinex5.Loader{}, l, &p)
	}
	if p.Total != 2 || p.Loaded != 2 || p.Mod != "MoreCompany 1.9.0" || !p.Ready {
		t.Fatalf("%+v", p)
	}
}

func TestThrottleProgressReportsTheFirstAndLastAndSkipsTheMiddle(t *testing.T) {
	var got [][2]int
	fn := throttleProgress(func(done, total int) { got = append(got, [2]int{done, total}) }, time.Hour)
	fn(0, 3)
	fn(1, 3)
	fn(2, 3)
	fn(3, 3)
	if len(got) != 2 || got[0] != [2]int{0, 3} || got[1] != [2]int{3, 3} {
		t.Fatalf("reports = %v, want the first and the last", got)
	}
}
