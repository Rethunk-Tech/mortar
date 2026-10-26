package game

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	gameruntime "github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/steam"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// StartEnv is what starting a plan needs beyond the plan itself.
type StartEnv struct {
	// Steam is nil when no Steam was found.
	Steam *steam.Steam
	// Direct starts the game's own process instead of asking its store to, after the user agreed to lose the
	// overlay and playtime.
	Direct bool
	// HideWindow hides the process window (Windows SMAPI console).
	HideWindow bool
	// LogFile is the loader's log, rewritten when the game starts; a profile launch succeeds when it is.
	LogFile string
	// Ready reports that the game process is running: a vanilla launch has then succeeded, and a profile launch whose
	// log is late has still started.
	Ready func() bool
	// OnStage reports each launch.Stage a start reaches; a launch of several processes reports each one's.
	OnStage func(launch.Stage)
	// OnVia reports the launcher a start hands the game to (launch.Command.Via), before it starts; never called for a
	// direct start.
	OnVia func(string)
	// OnExit reports how the started process ended once the game has started. Not used for relays.
	OnExit func(launch.Exit)
	// Proton is the Proton build (its folder name) a Windows copy outside Steam runs through; empty takes the best.
	Proton string
	// Instances is how many processes a direct Play starts (1–8). Steam and store URL launches always start one.
	Instances int
	// DelaySeconds is the pause between those processes. Test copies start without it: the dev helper's joiners retry
	// until the host is up.
	DelaySeconds int
	// TestCopies gives the first process `--lan --host` and the others `--lan --join 127.0.0.1`, which a game's dev
	// helper mod reads to host a LAN game and join it, and each `--tile <n>/<count>`, its cell of a grid of windows.
	TestCopies bool
	// StartFirst is the VR runtime started before the game (settings.StartFirstSteamVR or StartFirstALVR), and
	// ALVRProgram ALVR's dashboard; empty or off starts none.
	StartFirst  string
	ALVRProgram string
}

// Starter starts launch plans the way the install's store requires: Steam installs through `steam -applaunch`, which
// hands the start to the game and exits, every other store by running the game's executable. The zero value is the
// real thing; the fields exist so tests can point it elsewhere.
type Starter struct {
	// Runner, LookPath and Timing default to the real command, exec.LookPath and launch.Timing's defaults.
	Runner   launch.Runner
	LookPath func(string) (string, error)
	Timing   launch.Timing
	// DataDir is Mortar's data folder, used to check Flatpak Steam filesystem access; empty means datadir.Dir.
	DataDir string
	// FlatpakShow is `flatpak override --user --show` for Steam; nil means the real command.
	FlatpakShow func() (string, error)
	// Sleep waits d or until ctx is done. Tests replace it so scheduling does not wait on a real clock.
	Sleep func(ctx context.Context, d time.Duration) error
}

// Start runs plan for inst and returns once the game has started, sending the loader's log lines to onLines in
// batches until ctx is done. It returns launch.ErrNoSteam when there is no Steam and env.Direct is false.
func (st Starter) Start(ctx context.Context, inst Install, plan *launchplan.Plan, env StartEnv, onLines func([]string)) error {
	n := 1
	if env.Direct {
		n = min(max(env.Instances, 1), 8)
	}
	sleep := st.Sleep
	if sleep == nil {
		sleep = waitDelay
	}
	if err := st.startVR(ctx, env); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if i > 0 && !env.TestCopies {
			d := max(env.DelaySeconds, 0)
			if err := sleep(ctx, time.Duration(d)*time.Second); err != nil {
				return err
			}
		}
		one := plan
		if env.TestCopies {
			role := []string{"--lan", "--join", "127.0.0.1"}
			if i == 0 {
				role = []string{"--lan", "--host"}
			}
			copied := *plan
			copied.Args = append(slices.Clone(plan.Args), append(role, "--tile", fmt.Sprintf("%d/%d", i+1, n))...)
			if i > 0 && plan.ForCopy != nil {
				copied.Env = maps.Clone(plan.Env)
				if err := plan.ForCopy(i+1, &copied); err != nil {
					return err
				}
			}
			one = &copied
		}
		if err := st.startOne(ctx, inst, one, env, onLines); err != nil {
			return err
		}
	}
	return nil
}

func waitDelay(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (st Starter) lookPathOrDefault() func(string) (string, error) {
	switch {
	case st.LookPath != nil:
		return st.LookPath
	case sandbox.InFlatpak():
		return sandbox.HostLookPath
	}
	return exec.LookPath
}

func (st Starter) startOne(ctx context.Context, inst Install, plan *launchplan.Plan, env StartEnv, onLines func([]string)) error {
	lookPath := st.lookPathOrDefault()
	steamPath, _ := lookPath("steam")
	flatpakPath, _ := lookPath("flatpak")
	cmd, err := st.command(runtime.GOOS, inst, plan, env, steamPath, flatpakPath)
	if err != nil {
		return err
	}
	if sandbox.InFlatpak() {
		cmd = onHost(cmd)
	}
	cmd.OnExit = env.OnExit
	cmd.OnStage = env.OnStage
	if cmd.Relay && env.OnVia != nil {
		env.OnVia(cmd.Via)
	}
	// A game whose loader writes no log (a folder game) is up when its process is.
	if plan.Mode == launchplan.ModeVanilla || env.LogFile == "" {
		cmd.Ready = env.Ready
	} else {
		cmd.LogFile = env.LogFile
		cmd.Alive = env.Ready
	}
	run := st.Runner
	if run == nil {
		run = func(dir, name string, args ...string) (<-chan error, error) {
			// A relay starts no game of its own, so only a direct start goes through the guard.
			if !cmd.Relay {
				name, args = guarded(launchWrapper(), name, args)
			}
			if env.HideWindow {
				return launch.StartHidden(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
			}
			return launch.StartWithEnv(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
		}
	}
	return launch.Run(ctx, run, cmd, st.Timing, onLines)
}

// guarded runs name through wrapper when there is one: a server build's self-test sandbox names its launch guard
// (scripts/launch-guard.sh), which caps, records and isolates every game it starts.
func guarded(wrapper, name string, args []string) (string, []string) {
	if wrapper == "" {
		return name, args
	}
	return wrapper, append([]string{name}, args...)
}

// Command builds the direct start of plan without running it.
func (st Starter) Command(goos string, inst Install, plan *launchplan.Plan) (launch.Command, error) {
	return st.command(goos, inst, plan, StartEnv{Direct: true}, "", "")
}

// onHost reruns cmd through flatpak-spawn: the sandbox has no Steam, and a direct launch must not inherit its
// permissions, which also spares the manifest an audio socket and --device=all.
func onHost(cmd launch.Command) launch.Command {
	cmd.Name, cmd.Args = sandbox.HostArgv(cmd.Dir, cmd.Env, cmd.Name, cmd.Args...)
	cmd.Dir, cmd.Env = "", nil
	return cmd
}

// steamSkipsLoader reports a Windows Steam start that would run the game without its loader: Steam starts the loader
// only when the game's launch options name it, which nothing but Mortar's own setup sets (Vortex and SMAPI's
// installer run the loader directly). Starting the loader directly then loads the mods and gives Mortar the process
// to follow, at the cost of the Steam overlay.
func steamSkipsLoader(goos string, inst Install, plan *launchplan.Plan, env StartEnv) bool {
	if goos != "windows" || env.Steam == nil || plan.Mode != launchplan.ModeProfile || plan.Entry == "" {
		return false
	}
	info, _ := catalogGame(inst.Game)
	opts, err := env.Steam.LaunchOptions(info.SteamAppID())
	return err != nil || !launchHas(opts, plan.Entry)
}

// command builds the process to start. steamPath and flatpakPath are LookPath results, "" when missing.
func (st Starter) command(goos string, inst Install, plan *launchplan.Plan, env StartEnv, steamPath, flatpakPath string) (launch.Command, error) {
	if inst.Store == StoreEpic && !env.Direct && plan.Exe == "" {
		return epicCommand(inst)
	}
	if inst.Launcher != "" && !env.Direct {
		if cmd, ok, err := launcherCommand(inst, plan, st.lookPathOrDefault()); err != nil || ok {
			return cmd, err
		}
		env.Direct = true
	}
	// A loader that replaces the executable cannot be reproduced by a store's relay.
	if env.Direct || plan.Exe != "" || steamSkipsLoader(goos, inst, plan, env) {
		if inst.Store == StoreFlatpak {
			return flatpakAppCommand(inst, plan, flatpakPath)
		}
		exe := cmp.Or(plan.Exe, plan.Entry)
		// A loader that injects into the game (BepInEx) names no executable of its own: the game's is the start.
		if info, ok := catalogGame(inst.Game); ok && exe == "" {
			switch {
			case inst.Platform == "linux" && info.LinuxMarker != "":
				exe = filepath.Join(inst.Dir, filepath.FromSlash(cmp.Or(info.LinuxStart, info.LinuxMarker)))
			case strings.HasSuffix(strings.ToLower(info.Marker), ".exe"):
				exe = filepath.Join(inst.Dir, info.MarkerPath())
			}
		}
		if exe == "" {
			return launch.Command{}, errors.New("the game has no executable to start")
		}
		args := slices.Clone(plan.Args)
		if inst.RunsInBottle() {
			argv, err := inst.BottleCommand(exe, args...)
			if err != nil {
				return launch.Command{}, err
			}
			return launch.Command{Dir: inst.Dir, Name: argv[0], Args: argv[1:], Env: environ(plan.Env)}, nil
		}
		var protonEnv []string
		if goos != "windows" && (inst.Runtime == gameruntime.MortarProton || inst.Runtime == gameruntime.LauncherPrefix ||
			(inst.Runtime == gameruntime.Proton && inst.Store == StoreSteam)) {
			info, _ := catalogGame(inst.Game)
			rt := inst.runtime(info, "")
			rt.Proton = env.Proton
			argv, penv, err := gameruntime.ProtonCommand(rt, exe, args...)
			if err != nil {
				return launch.Command{}, err
			}
			// waitforexitandrun waits for everything else in the prefix to end, so a second copy would wait for the first.
			if env.Instances > 1 && len(argv) > 1 && argv[1] == "waitforexitandrun" {
				argv[1] = "run"
			}
			exe, args, protonEnv = argv[0], argv[1:], penv
		}
		name := exe
		if goos != "windows" && len(plan.Prefix) > 0 {
			name, args = plan.Prefix[0], append(slices.Clone(plan.Prefix[1:]), append([]string{exe}, args...)...)
		}
		env := append(environ(plan.Env), protonEnv...)
		if goos == "linux" && inst.Platform == "linux" {
			env = append(env, dotnetRoot(plan.Env)...)
		}
		// Wine prints its errors only when asked; without them a launch that stalls on one says nothing.
		if (inst.Runtime == gameruntime.Proton || inst.Runtime == gameruntime.MortarProton || inst.Runtime == gameruntime.LauncherPrefix) && plan.Env["WINEDEBUG"] == "" {
			env = append(env, "WINEDEBUG=err+all")
		}
		return launch.Command{Dir: inst.Dir, Name: name, Args: args, Env: env}, nil
	}
	info, _ := catalogGame(inst.Game)
	return st.steamRelay(goos, env, steamPath, flatpakPath, append([]string{"-applaunch", info.SteamAppID()}, plan.Args...))
}

// steamRelay hands args to Steam (`steam -applaunch <app>`), which starts the app and exits.
func (st Starter) steamRelay(goos string, env StartEnv, steamPath, flatpakPath string, args []string) (launch.Command, error) {
	switch {
	case env.Steam == nil:
		return launch.Command{}, launch.ErrNoSteam
	case goos == "windows":
		return launch.Command{Name: filepath.Join(env.Steam.Root, "steam.exe"), Args: args, Failure: launch.HintSteam, Relay: true, Via: "steam"}, nil
	case env.Steam.Kind == steam.KindFlatpak:
		if flatpakPath == "" {
			return launch.Command{}, launch.ErrNoSteam
		}
		hint := launch.HintSteam
		if !st.flatpakCanReadMods() {
			hint = launch.HintFlatpakFS
		}
		return launch.Command{Name: flatpakPath, Args: append([]string{"run", steam.FlatpakID}, args...), Failure: hint, Relay: true, Via: "flatpak-steam"}, nil
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{Name: steamPath, Args: args, Failure: launch.HintSteam, Relay: true, Via: "steam"}, nil
}

// flatpakAppCommand starts a game installed as a Flatpak app through `flatpak run`, which stays until the game exits.
func flatpakAppCommand(inst Install, plan *launchplan.Plan, flatpakPath string) (launch.Command, error) {
	info, _ := catalogGame(inst.Game)
	if info.Stores.Flatpak == nil {
		return launch.Command{}, errors.New("the game has no Flatpak app")
	}
	if flatpakPath == "" {
		return launch.Command{}, errors.New("flatpak is not installed, and this copy of the game is a Flatpak app")
	}
	args := []string{"run"}
	for _, kv := range environ(plan.Env) {
		args = append(args, "--env="+kv)
	}
	args = append(append(args, info.Stores.Flatpak.App), plan.Args...)
	return launch.Command{Name: flatpakPath, Args: args}, nil
}

// dotnetRoot points a .NET build at the runtime the .NET install script puts in ~/.dotnet, which a game started from
// the desktop does not otherwise find: the script sets DOTNET_ROOT only in the shell's profile.
func dotnetRoot(planEnv map[string]string) []string {
	if os.Getenv("DOTNET_ROOT") != "" || planEnv["DOTNET_ROOT"] != "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil || !fsx.IsFile(filepath.Join(home, ".dotnet", "dotnet")) {
		return nil
	}
	return []string{"DOTNET_ROOT=" + filepath.Join(home, ".dotnet")}
}

func (st Starter) flatpakCanReadMods() bool {
	show := st.FlatpakShow
	if show == nil {
		show = steam.ShowOverride
	}
	out, err := show()
	if err != nil {
		return false
	}
	dir := st.DataDir
	if dir == "" {
		dir, err = datadir.Dir()
		if err != nil {
			return false
		}
	}
	return steam.HasFilesystem(out, dir)
}

// environ is env as KEY=VALUE pairs in key order; nil when there are none.
func environ(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	slices.Sort(out)
	return out
}

// steamVRAppID is SteamVR's Steam app.
const steamVRAppID = "250820"

// startVR starts the VR runtime the game is set to start first and returns without waiting for it: SteamVR through
// Steam, or ALVR's dashboard (which opens SteamVR itself) unless it already runs. The game then starts as usual.
func (st Starter) startVR(ctx context.Context, env StartEnv) error {
	var cmd launch.Command
	switch env.StartFirst {
	case settings.StartFirstSteamVR:
		lookPath := st.lookPathOrDefault()
		steamPath, _ := lookPath("steam")
		flatpakPath, _ := lookPath("flatpak")
		c, err := st.steamRelay(runtime.GOOS, env, steamPath, flatpakPath, []string{"-applaunch", steamVRAppID})
		if err != nil {
			return err
		}
		cmd = c
	case settings.StartFirstALVR:
		if env.ALVRProgram == "" {
			return usererr.Wrap(usererr.Invalid, errors.New("ALVR is set to start first, but its program is not chosen: choose it in the game's Play settings"))
		}
		if running, err := launch.Processes("/proc", filepath.Base(env.ALVRProgram)); err == nil && len(running) > 0 {
			return nil
		}
		cmd = launch.Command{Dir: filepath.Dir(env.ALVRProgram), Name: env.ALVRProgram}
	default:
		return nil
	}
	if sandbox.InFlatpak() {
		cmd = onHost(cmd)
	}
	run := st.Runner
	if run == nil {
		run = func(dir, name string, args ...string) (<-chan error, error) {
			return launch.StartWithEnv(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
		}
	}
	if _, err := run(cmd.Dir, cmd.Name, cmd.Args...); err != nil {
		return fmt.Errorf("start %s first: %w", env.StartFirst, err)
	}
	return nil
}
