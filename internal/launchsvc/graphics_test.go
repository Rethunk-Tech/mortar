package launchsvc

import (
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

var testOffer = &components.Graphics{
	Recommended: "dx12",
	Choices:     []components.GraphicsChoice{{ID: "dx12", Label: "DX12", Args: []string{"-dx12"}}, {ID: "vulkan", Label: "Vulkan"}},
}

func TestChosenGraphicsResolvesGameThenProfile(t *testing.T) {
	var st settings.Settings
	sc := settings.Scope{Game: "peak"}
	if _, ok := chosenGraphics(testOffer, st, sc, nil); ok {
		t.Fatal("an unset setting chose something")
	}
	if err := settings.ApplyKeyGame(&st, "graphicsApi", "dx12", "peak"); err != nil {
		t.Fatal(err)
	}
	if c, ok := chosenGraphics(testOffer, st, sc, nil); !ok || !slices.Equal(c.Args, []string{"-dx12"}) {
		t.Errorf("game choice = %+v, %v", c, ok)
	}
	if c, ok := chosenGraphics(testOffer, st, sc, map[string]string{"graphicsApi": "vulkan"}); !ok || len(c.Args) != 0 {
		t.Errorf("profile override = %+v, %v; want Vulkan with no arguments", c, ok)
	}
	if _, ok := chosenGraphics(testOffer, st, sc, map[string]string{"graphicsApi": "gone"}); ok {
		t.Error("an id the catalog lacks chose something")
	}
	if _, ok := chosenGraphics(nil, st, sc, nil); ok {
		t.Error("a game with no offer chose something")
	}
}

func TestGraphicsMarkerAsksAgainOncePerFailureThenStops(t *testing.T) {
	early, lasted := earlyExitWindow-time.Second, earlyExitWindow
	m := (graphicsMarker{}).afterRun(early)
	if !m.Pending {
		t.Fatal("an early exit did not ask on the next Play")
	}
	if m = m.afterAnswer(false); m.Pending || m.Asked != 1 {
		t.Fatalf("keeping Vulkan = %+v, want the question spent", m)
	}
	if m = m.afterRun(early); !m.Pending {
		t.Fatalf("a second early exit = %+v, want one more question", m)
	}
	if m = m.afterAnswer(false).afterRun(early); m.Pending {
		t.Fatalf("a third early exit = %+v, want no more questions", m)
	}
	if m = m.afterRun(lasted); m != (graphicsMarker{}) {
		t.Errorf("a run that lasted left %+v", m)
	}
	if m = (graphicsMarker{Pending: true, Asked: 1}).afterAnswer(true); m != (graphicsMarker{}) {
		t.Errorf("choosing the recommendation left %+v", m)
	}
}

func TestAnEarlyExitUnderVulkanBringsBackPlaysQuestion(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "peak", "A")
	svc := NewService(t.TempDir(), set, profiles)
	g := game.Find("peak")
	choose := func(id string) {
		t.Helper()
		if _, err := set.Update(func(v *settings.Settings) {
			if err := settings.ApplyKeyGame(v, "graphicsApi", id, "peak"); err != nil {
				t.Fatal(err)
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	ask := func() GraphicsAsk {
		t.Helper()
		a, err := svc.GraphicsAsk("peak", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	if a := ask(); !a.Ask || a.AfterEarlyExit {
		t.Fatalf("unset = %+v, want the plain question", a)
	}
	choose("vulkan")
	if a := ask(); a.Ask {
		t.Fatalf("answered = %+v", a)
	}
	svc.noteGraphicsExit(g, p.ID, 5*time.Second)
	if a := ask(); !a.Ask || !a.AfterEarlyExit {
		t.Fatalf("after an early exit = %+v", a)
	}
	if err := svc.GraphicsAnswered("peak", p.ID, "vulkan"); err != nil {
		t.Fatal(err)
	}
	if a := ask(); a.Ask {
		t.Fatalf("after keeping Vulkan = %+v, want the question spent", a)
	}
	choose("dx12")
	svc.noteGraphicsExit(g, p.ID, 5*time.Second)
	if a := ask(); a.Ask {
		t.Fatalf("an early exit under the recommendation = %+v, want no question", a)
	}
}

func peakService(t *testing.T) (*Service, *settings.Store, string) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "peak", "A")
	return NewService(t.TempDir(), set, profiles), set, p.ID
}

func TestRecordedRunsMarkOrClearTheGraphicsRecord(t *testing.T) {
	svc, set, id := peakService(t)
	if _, err := set.Update(func(v *settings.Settings) {
		if err := settings.ApplyKeyGame(v, "graphicsApi", "vulkan", "peak"); err != nil {
			t.Fatal(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	g := game.Find("peak")
	pending := func() bool {
		t.Helper()
		a, err := svc.GraphicsAsk("peak", id)
		if err != nil {
			t.Fatal(err)
		}
		return a.AfterEarlyExit
	}
	svc.record(g, id, time.Now().Add(-5*time.Second), false)
	if !pending() {
		t.Fatal("a run that ended 5s after Play under Vulkan did not mark the profile")
	}
	svc.record(g, id, time.Now().Add(-earlyExitWindow-time.Second), false)
	if pending() {
		t.Fatal("a run that lasted past the window left the mark")
	}
}

func TestLaunchPlanCarriesTheChosenGraphicsArguments(t *testing.T) {
	svc, set, id := peakService(t)
	dir, err := svc.profiles.ProfileDir("peak", id)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"winhttp.dll", "doorstop_config.ini"} {
		testfs.WriteFile(t, dir, f, "x")
	}
	install := t.TempDir()
	testfs.WriteFile(t, install, "PEAK.exe", "exe")
	g := game.Find("peak")
	inst := game.Install{ID: "peak1", Dir: install}
	args := func() []string {
		t.Helper()
		plan, err := svc.launchPlan(t.Context(), g, inst, id, launchplan.ModeProfile, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		return plan.Args
	}
	if slices.Contains(args(), "-dx12") {
		t.Fatal("an unset choice passed -dx12")
	}
	if _, err := set.Update(func(v *settings.Settings) {
		if err := settings.ApplyKeyGame(v, "graphicsApi", "dx12", "peak"); err != nil {
			t.Fatal(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(args(), "-dx12") {
		t.Fatalf("args = %v, want -dx12 for DirectX 12", args())
	}
	if _, err := svc.profiles.SetOverrides("peak", id, map[string]string{"graphicsApi": "vulkan"}); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(args(), "-dx12") {
		t.Fatalf("args = %v, want no -dx12 under the profile's Vulkan override", args())
	}
}
