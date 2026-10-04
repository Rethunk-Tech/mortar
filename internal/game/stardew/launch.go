package stardew

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// ProcessName is the executable of the running loader.
func (Game) ProcessName() string { return smapiMarker }

// GameProcesses covers SMAPI, the vanilla game's native executable, and the launcher names either install leaves.
func (Game) GameProcesses() []string {
	// StardewValley-original execs the native binary; vanilla success watches that name too.
	return []string{smapiMarker, "Stardew Valley", linuxLauncher, linuxOriginal, "StardewValley.bin.x86_64"}
}

// LogFile is SMAPI-latest.txt, which SMAPI rewrites on each start.
func (g Game) LogFile() (string, error) {
	dir, err := g.logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SMAPI-latest.txt"), nil
}

// Launch starts SMAPI on req.ModsDir and returns once SMAPI has rewritten its log, sending the log's lines to
// onLines until ctx is done. It returns launch.ErrNoSteam when there is no Steam to launch through and req.Direct is false.
func (g Game) Launch(ctx context.Context, req launch.Request, onLines func([]string)) error {
	lookPath := g.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
		if sandbox.InFlatpak() {
			lookPath = sandbox.HostLookPath
		}
	}
	steamPath, _ := lookPath("steam")
	flatpakPath, _ := lookPath("flatpak")
	cmd, err := g.command(runtime.GOOS, req, steamPath, flatpakPath)
	if err != nil {
		return err
	}
	if sandbox.InFlatpak() {
		cmd = onHost(cmd)
	}
	cmd.OnExit = req.OnExit
	if req.Vanilla {
		cmd.Ready = req.Seen
	} else {
		logFile, err := g.LogFile()
		if err != nil {
			return err
		}
		cmd.LogFile = logFile
	}
	run := g.Runner
	if run == nil {
		run = func(dir, name string, args ...string) (<-chan error, error) {
			if req.HideWindow {
				return launch.StartHidden(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
			}
			return launch.StartWithEnv(context.WithoutCancel(ctx), cmd.Env, dir, name, args...)
		}
	}
	return launch.Run(ctx, run, cmd, g.LaunchTiming, onLines)
}

// onHost reruns cmd through flatpak-spawn: the sandbox has no Steam, and a direct launch must not inherit its
// permissions, which also spares the manifest an audio socket and --device=all.
func onHost(cmd launch.Command) launch.Command {
	cmd.Name, cmd.Args = sandbox.HostArgv(cmd.Dir, cmd.Env, cmd.Name, cmd.Args...)
	cmd.Dir, cmd.Env = "", nil
	return cmd
}

// DirectCommand builds the command used for a direct profile launch without starting it.
func (g Game) DirectCommand(goos string, req launch.Request) (launch.Command, error) {
	req.Direct = true
	return g.command(goos, req, "", "")
}

// command builds the process to start. steamPath and flatpakPath are LookPath results, "" when missing.
func (g Game) command(goos string, req launch.Request, steamPath, flatpakPath string) (launch.Command, error) {
	if req.Vanilla {
		return g.vanillaCommand(goos, req)
	}
	if !filepath.IsAbs(req.ModsDir) {
		return launch.Command{}, fmt.Errorf("mods folder %q is not an absolute path", req.ModsDir)
	}
	modsArgs := []string{"--mods-path", req.ModsDir}
	smapiArgs := appendLaunchArgs(goos, modsArgs, req.ExtraArgs)
	windows := goos == "windows"
	if req.Direct {
		directArgs := smapiArgs
		if !windows {
			directArgs = append(append([]string{}, req.Prefix...), smapiArgs...)
		}
		if windows {
			return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, smapiMarker+".exe"), Args: directArgs, Env: req.Env}, nil
		}
		// The launcher reads its own flags before `--` and forwards only what follows it.
		return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, linuxLauncher), Args: directArgs, Env: req.Env}, nil
	}
	appID := []string{"-applaunch", g.SteamAppID()}
	switch {
	case req.Steam == nil:
		return launch.Command{}, launch.ErrNoSteam
	case windows:
		hint := launch.HintSteam
		if opts, err := req.Steam.LaunchOptions(g.SteamAppID()); err == nil && !hasSMAPILine(opts) {
			hint = launch.HintLaunchOptions
		}
		return launch.Command{Name: filepath.Join(req.Steam.Root, "steam.exe"), Args: append(appID, smapiArgs...), Failure: hint, Relay: true}, nil
	case req.Steam.Kind == steam.KindFlatpak:
		if flatpakPath == "" {
			return launch.Command{}, launch.ErrNoSteam
		}
		hint := launch.HintSteam
		if !g.flatpakCanReadMods() {
			hint = launch.HintFlatpakFS
		}
		args := append([]string{"run", steam.FlatpakID}, append(appID, smapiArgs...)...)
		return launch.Command{Name: flatpakPath, Args: args, Failure: hint, Relay: true}, nil
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{Name: steamPath, Args: append(appID, smapiArgs...), Failure: launch.HintSteam, Relay: true}, nil
}

func (g Game) flatpakCanReadMods() bool {
	show := g.FlatpakShow
	if show == nil {
		show = steam.ShowOverride
	}
	out, err := show()
	if err != nil {
		return false
	}
	dir := g.DataDir
	if dir == "" {
		dir, err = datadir.Dir()
		if err != nil {
			return false
		}
	}
	return steam.HasFilesystem(out, dir)
}

func (g Game) vanillaCommand(goos string, req launch.Request) (launch.Command, error) {
	windows := goos == "windows"
	if !windows || req.Direct {
		name := "Stardew Valley.exe"
		if !windows {
			// SMAPI's unix-launcher.sh always execs StardewModdingAPI; steam -applaunch hits that
			// script. There is no vanilla flag. The installer kept the unmodded game as StardewValley-original.
			name = linuxOriginal
		}
		return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, name)}, nil
	}
	if req.Steam == nil {
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{
		Name:    filepath.Join(req.Steam.Root, "steam.exe"),
		Args:    []string{"-applaunch", g.SteamAppID()},
		Failure: launch.HintSteam,
		Relay:   true,
	}, nil
}

// SteamLaunchForcesLoader reports whether Steam launch options run SMAPI in place of the game.
func (Game) SteamLaunchForcesLoader(options string) bool {
	return hasSMAPILine(options)
}

// SteamLaunchWithLoader puts SMAPI before Steam's %command%, or adds the SMAPI line ahead of options without one.
func (Game) SteamLaunchWithLoader(dir, current string) string {
	if hasSMAPILine(current) {
		return current
	}
	exe := `"` + filepath.Join(dir, smapiMarker+".exe") + `"`
	if strings.Contains(current, "%command%") {
		return strings.Replace(current, "%command%", exe+" %command%", 1)
	}
	return strings.TrimSpace(exe + " %command% " + current)
}

// SteamLaunchWithoutLoader removes SMAPI before Steam's %command%, keeping the user's other options.
func (Game) SteamLaunchWithoutLoader(current string) string {
	if !hasSMAPILine(current) {
		if strings.TrimSpace(current) == "%command%" {
			return ""
		}
		return current
	}
	prefix, suffix, ok := strings.Cut(current, "%command%")
	if !ok {
		return current
	}
	closeQuote := strings.LastIndex(prefix, `"`)
	if closeQuote < 0 {
		return current
	}
	openQuote := strings.LastIndex(prefix[:closeQuote], `"`)
	if openQuote < 0 || !strings.Contains(strings.ToLower(prefix[openQuote:closeQuote+1]), strings.ToLower(smapiMarker)) {
		return current
	}
	result := prefix[:openQuote] + "%command%" + suffix
	result = strings.TrimSpace(result)
	if result == "%command%" {
		return ""
	}
	return result
}

// hasSMAPILine reports whether Steam launch options run SMAPI in place of the game: `"<game>\StardewModdingAPI.exe" %command%`.
func hasSMAPILine(options string) bool {
	return strings.Contains(strings.ToLower(options), strings.ToLower(smapiMarker)) && strings.Contains(options, "%command%")
}
