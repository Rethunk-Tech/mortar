package bepinex5

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestLoaderContract(t *testing.T) {
	ctx := t.Context()
	l, ok := loader.Get("bepinex5")
	if !ok {
		t.Fatal("bepinex5 is not registered")
	}
	if got := loader.For(components.GameInfo{Loaders: []components.GameLoader{{ID: "nope"}, {ID: "bepinex5"}}}); len(got) != 1 || got[0].ID() != ID {
		t.Fatalf("For: %v", got)
	}
	tgt := loader.Target{ProfileDir: filepath.Join(t.TempDir(), "profile")}
	if st, err := l.Status(tgt); err != nil || st.Installed {
		t.Fatalf("fresh status %+v %v", st, err)
	}
	var steps []loader.Step
	ver, err := l.Install(ctx, tgt, loader.Package{Archive: buildPack(t, "4.3.0.0")}, func(s loader.Step) { steps = append(steps, s) })
	if err != nil || ver != "5.4.2305" || len(steps) != 1 {
		t.Fatalf("install %q %v %v", ver, steps, err)
	}
	if st, _ := l.Status(tgt); !st.Installed || st.Version != "5.4.2305" {
		t.Fatalf("status %+v", st)
	}

	view := loader.ProfileView{Dir: tgt.ProfileDir, Runtime: "proton"}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(ctx, plan, view); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.Args, " "); !strings.HasPrefix(got, `--doorstop-enabled true --doorstop-target-assembly Z:\`) {
		t.Fatalf("args %q", got)
	}
	if len(plan.Files) != 2 || plan.Files[0].Dst != "doorstop_config.ini" || len(plan.RuntimeReqs) != 1 {
		t.Fatalf("plan %+v", plan)
	}
	native := launchplan.New(launchplan.ModeProfile)
	view.Runtime = "native"
	_ = l.Contribute(ctx, native, view)
	if len(native.RuntimeReqs) != 0 || strings.Contains(strings.Join(native.Args, " "), "Z:") {
		t.Fatalf("native plan %+v", native)
	}

	van := launchplan.New(launchplan.ModeVanilla)
	vanilla, ok := l.(loader.Vanilla)
	if !ok {
		t.Fatal("not a Vanilla loader")
	}
	if err := vanilla.Vanilla(ctx, van, tgt); err != nil || strings.Join(van.Args, " ") != "--doorstop-enabled false" {
		t.Fatalf("vanilla %v %v", van.Args, err)
	}
	logs, ok := l.(loader.WithLogs)
	if !ok {
		t.Fatal("not a WithLogs loader")
	}
	if p, _ := logs.Path(view); !strings.HasSuffix(p, filepath.Join("BepInEx", "LogOutput.log")) || !logs.Ready("[Message:   BepInEx] Chainloader startup complete") {
		t.Fatalf("logs %q", p)
	}
	found := logs.Analyzers()[0].Analyze(loader.Logs{Loader: "[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B\n"})
	if len(found) != 1 || found[0].Kind != KindMissingDependency {
		t.Fatalf("findings %+v", found)
	}
	cfg, isCfg := l.(loader.WithConfig)
	if _, ok := l.(loader.WinHTTPLoader); !ok || !isCfg || cfg.ConfigDirs()[0] != "BepInEx/config" {
		t.Fatal("capabilities")
	}
}

func TestQueryReadsTheProfilesStateFile(t *testing.T) {
	l := Loader{}
	var _ loader.Querier = l
	var _ loader.WithCompanion = l
	p := loader.ProfileView{Dir: t.TempDir()}
	if _, err := l.Query(t.Context(), loader.Target{}, p, "status"); !errors.Is(err, bridge.ErrNotReady) {
		t.Fatalf("a profile whose game has not started: err = %v", err)
	}
	if l.Companion().ID != "Rethunk.MortarBepInExBridge" {
		t.Fatalf("companion = %+v", l.Companion())
	}
}
