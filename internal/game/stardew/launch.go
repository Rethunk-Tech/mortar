package stardew

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/steam"
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
	}
	steamPath, _ := lookPath("steam")
	flatpakPath, _ := lookPath("flatpak")
	cmd, err := g.command(runtime.GOOS, req, steamPath, flatpakPath)
	if err != nil {
		return err
	}
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
			return launch.StartWithEnv(cmd.Env, dir, name, args...)
		}
	}
	return launch.Run(ctx, run, cmd, g.LaunchTiming, onLines)
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

// hasSMAPILine reports whether Steam launch options run SMAPI in place of the game: `"<game>\StardewModdingAPI.exe" %command%`.
func hasSMAPILine(options string) bool {
	return strings.Contains(strings.ToLower(options), strings.ToLower(smapiMarker)) && strings.Contains(options, "%command%")
}
