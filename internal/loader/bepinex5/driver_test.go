package bepinex5

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/serverargs"
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
	// The game folder's config names the profile's preloader itself, for a start that passes no Doorstop flags.
	if cfg, err := os.ReadFile(plan.Files[0].Src); err != nil || !strings.Contains(string(cfg), "target_assembly=Z:\\") {
		t.Fatalf("config %q %v", cfg, err)
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
	if err := vanilla.Vanilla(ctx, van, tgt); err != nil || strings.Join(van.Args, " ") != "--doorstop-enable false --doorstop-enabled false" {
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
	if !isCfg || cfg.ConfigDirs()[0] != "BepInEx/config" {
		t.Fatal("capabilities")
	}
}

func TestQueryReadsTheProfilesStateFile(t *testing.T) {
	l := Loader{}
	var _ loader.WithCompanion = l
	p := loader.ProfileView{Dir: t.TempDir()}
	if _, err := l.Query(t.Context(), loader.Target{}, p, "status"); !errors.Is(err, bridge.ErrNotReady) {
		t.Fatalf("a profile whose game has not started: err = %v", err)
	}
	if l.Companion().ID != "Rethunk.MortarBepInExBridge" {
		t.Fatalf("companion = %+v", l.Companion())
	}
}

func TestANativeLinuxBuildStartsDirectlyWithDoorstopPreloaded(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	if _, err := (Loader{}).Install(t.Context(), loader.Target{ProfileDir: profile}, loader.Package{Archive: buildPackIn(t, "BepInExPack_Valheim", "4.4.0")}, nil); err != nil {
		t.Fatal(err)
	}
	view := loader.ProfileView{Game: "valheim", Dir: profile, InstallDir: "/games/Valheim", Runtime: "native", Platform: "linux"}
	if err := (Loader{}).Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), view); err == nil {
		t.Fatal("a pack without a Linux Doorstop library started")
	}
	lib := filepath.Join(profile, "doorstop_libs", "libdoorstop_x64.so")
	if err := fsx.WriteFile(lib, []byte("so"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), plan, view); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(profile, "BepInEx", "core", "BepInEx.Preloader.dll")
	if plan.Exe != "/games/Valheim/valheim.x86_64" || len(plan.Files) != 0 || len(plan.RuntimeReqs) != 0 {
		t.Fatalf("plan %+v", plan)
	}
	for k, want := range map[string]string{"LD_PRELOAD": lib, "DOORSTOP_ENABLED": "1", "DOORSTOP_TARGET_ASSEMBLY": target, "SteamAppId": "892970"} {
		if plan.Env[k] != want {
			t.Errorf("%s = %q, want %q", k, plan.Env[k], want)
		}
	}
	if !(Loader{}).Owns(loader.Process{Args: append([]string{plan.Exe}, plan.Args...)}, view) {
		t.Fatal("the profile does not own its own native process")
	}
}

func TestSendHandsTheLineToTheBridgeConsole(t *testing.T) {
	var _ loader.Console = Loader{}
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		r := bufio.NewReader(c)
		token, _ := r.ReadString('\n')
		line, _ := r.ReadString('\n')
		got <- token + line
		_, _ = io.WriteString(c, "ok\n")
	}()
	dir := t.TempDir()
	state := filepath.Join(dir, "BepInEx", "config", "mortar-bepinex-bridge.json")
	if err := os.MkdirAll(filepath.Dir(state), 0o750); err != nil {
		t.Fatal(err)
	}
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("listener has no TCP address")
	}
	port := tcp.Port
	if err := os.WriteFile(state, []byte(fmt.Sprintf(`{"port":%d,"token":"tok"}`, port)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Loader{}).Send(t.Context(), loader.Target{}, loader.ProfileView{Dir: dir}, "Spawn Boar"); err != nil {
		t.Fatal(err)
	}
	if line := <-got; line != "tok\nconsole Spawn Boar\n" {
		t.Fatalf("bridge got %q", line)
	}
}

func TestADedicatedServerLaunchesWithItsProfilesSettings(t *testing.T) {
	dir := t.TempDir()
	view := loader.ProfileView{Game: "valheim-server", Dir: dir, Runtime: "proton"}
	if err := (Loader{}).Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), view); err == nil || strings.Contains(err.Error(), "hunter22") {
		t.Fatalf("a server with no password set launched: %v", err)
	}
	g, _ := components.Game("valheim-server")
	for id, v := range map[string]string{"world": "Mine", "port": "2500", "password": "hunter22"} {
		if err := serverargs.Set(dir, *g.Server, id, v); err != nil {
			t.Fatal(err)
		}
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), plan, view); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(plan.Args, " ")
	for _, want := range []string{"-batchmode -nographics", "-name Mortar server", "-port 2500", "-world Mine", "-password hunter22", "-public 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("args %q lack %q", got, want)
		}
	}
	client := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), client, loader.ProfileView{Game: "valheim", Dir: t.TempDir(), Runtime: "proton"}); err != nil || strings.Contains(strings.Join(client.Args, " "), "-world") {
		t.Fatalf("the client got server arguments: %v %v", client.Args, err)
	}
}

func TestARiskOfRainServerIsConfiguredByFile(t *testing.T) {
	dir := t.TempDir()
	g, _ := components.Game("riskofrain2-server")
	if err := serverargs.Set(dir, *g.Server, "password", "open sesame"); err != nil {
		t.Fatal(err)
	}
	if err := serverargs.Set(dir, *g.Server, "port", "27020"); err != nil {
		t.Fatal(err)
	}
	if err := serverargs.Set(dir, *g.Server, "name", `bad";name`); err == nil {
		t.Fatal("a name that breaks out of its quotes was accepted")
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), plan, loader.ProfileView{Game: "riskofrain2-server", Dir: dir, Runtime: "proton"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plan.Args, " "), "sv_") {
		t.Fatalf("cvars on the command line: %v", plan.Args)
	}
	i := slices.IndexFunc(plan.Files, func(f launchplan.PlanFile) bool { return strings.HasSuffix(f.Dst, "server.cfg") })
	if i < 0 || plan.Files[i].Dst != filepath.FromSlash("Risk of Rain 2_Data/Config/server.cfg") {
		t.Fatalf("files %+v", plan.Files)
	}
	b, err := os.ReadFile(plan.Files[i].Src)
	if err != nil || !strings.Contains(string(b), "sv_port 27020;") || !strings.Contains(string(b), `sv_password "open sesame";`) {
		t.Fatalf("server.cfg = %q, %v", b, err)
	}
}

func TestLoadReadsTheChainloaderCountAndEachPlugin(t *testing.T) {
	for line, want := range map[string]struct {
		mod   string
		total int
	}{
		"[Info   :   BepInEx] 43 plugins to load":                {"", 43},
		"[Info   :   BepInEx] Loading [LCMaxSoundsFix 1.2.0]":    {"LCMaxSoundsFix 1.2.0", 0},
		"[Message:   BepInEx] Chainloader startup complete":      {"", 0},
		"[Info   : LethalLib] Loading [not the chainloader] now": {"", 0},
	} {
		if mod, total := (Loader{}).Load(line); mod != want.mod || total != want.total {
			t.Errorf("%q: got %q, %d", line, mod, total)
		}
	}
}
