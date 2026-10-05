package game

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/steam"
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
	// Ready reports that a vanilla launch succeeded, when the game process is running.
	Ready func() bool
	// OnExit reports how the started process ended once the game has started. Not used for relays.
	OnExit func(launch.Exit)
}

// Starter starts launch plans the way the install's store requires: Steam installs through `steam -applaunch`, which
// hands the start to the game and exits, every other store by running the game's executable. The zero value is the
// real thing; the fields exist so tests can point it elsewhere.
type Starter struct {
	// Runner, LookPath and Timing default to the real command, exec.LookPath and 60 s.
	Runner   launch.Runner
	LookPath func(string) (string, error)
	Timing   launch.Timing
	// DataDir is Mortar's data folder, used to check Flatpak Steam filesystem access; empty means datadir.Dir.
	DataDir string
	// FlatpakShow is `flatpak override --user --show` for Steam; nil means the real command.
	FlatpakShow func() (string, error)
}

// Start runs plan for inst and returns once the game has started, sending the loader's log lines to onLines in
// batches until ctx is done. It returns launch.ErrNoSteam when there is no Steam and env.Direct is false.
func (st Starter) Start(ctx context.Context, inst Install, plan *launchplan.Plan, env StartEnv, onLines func([]string)) error {
	lookPath := st.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
		if sandbox.InFlatpak() {
			lookPath = sandbox.HostLookPath
		}
	}
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
	if plan.Mode == launchplan.ModeVanilla {
		cmd.Ready = env.Ready
	} else {
		cmd.LogFile = env.LogFile
	}
	run := st.Runner
	if run == nil {
		run = func(dir, name string, args ...string) (<-chan error, error) {
			if env.HideWindow {
				return launch.StartHidden(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
			}
			return launch.StartWithEnv(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
		}
	}
	return launch.Run(ctx, run, cmd, st.Timing, onLines)
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

// command builds the process to start. steamPath and flatpakPath are LookPath results, "" when missing.
func (st Starter) command(goos string, inst Install, plan *launchplan.Plan, env StartEnv, steamPath, flatpakPath string) (launch.Command, error) {
	// A loader that replaces the executable cannot be reproduced by a store's relay.
	if env.Direct || plan.Exe != "" {
		exe := plan.Exe
		if exe == "" {
			exe = plan.Entry
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
		if goos != "windows" {
			args = append(slices.Clone(plan.Prefix), args...)
		}
		return launch.Command{Dir: inst.Dir, Name: exe, Args: args, Env: environ(plan.Env)}, nil
	}
	info, _ := catalogGame(inst.Game)
	appID := []string{"-applaunch", info.SteamAppID()}
	args := append(slices.Clone(appID), plan.Args...)
	switch {
	case env.Steam == nil:
		return launch.Command{}, launch.ErrNoSteam
	case goos == "windows":
		hint := launch.HintSteam
		if plan.Mode == launchplan.ModeProfile && plan.Entry != "" {
			if opts, err := env.Steam.LaunchOptions(info.SteamAppID()); err == nil && !launchHas(opts, plan.Entry) {
				hint = launch.HintLaunchOptions
			}
		}
		return launch.Command{Name: filepath.Join(env.Steam.Root, "steam.exe"), Args: args, Failure: hint, Relay: true}, nil
	case env.Steam.Kind == steam.KindFlatpak:
		if flatpakPath == "" {
			return launch.Command{}, launch.ErrNoSteam
		}
		hint := launch.HintSteam
		if !st.flatpakCanReadMods() {
			hint = launch.HintFlatpakFS
		}
		return launch.Command{Name: flatpakPath, Args: append([]string{"run", steam.FlatpakID}, args...), Failure: hint, Relay: true}, nil
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{Name: steamPath, Args: args, Failure: launch.HintSteam, Relay: true}, nil
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
	sort.Strings(out)
	return out
}
