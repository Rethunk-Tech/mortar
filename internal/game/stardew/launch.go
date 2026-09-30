package stardew

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

// ProcessName is the executable of the running loader.
func (Game) ProcessName() string { return smapiMarker }

// GameProcesses covers SMAPI, the vanilla game's native executable, and the launcher names either install leaves.
func (Game) GameProcesses() []string {
	return []string{smapiMarker, "Stardew Valley", linuxLauncher, linuxOriginal}
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
	cmd, err := g.command(runtime.GOOS, req, steamPath)
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
		run = launch.Start
	}
	return launch.Run(ctx, run, cmd, g.LaunchTiming, onLines)
}

// command builds the process to start. steamPath is the `steam` found on PATH, "" when there is none.
func (g Game) command(goos string, req launch.Request, steamPath string) (launch.Command, error) {
	if req.Vanilla {
		return g.vanillaCommand(goos, req, steamPath)
	}
	if !filepath.IsAbs(req.ModsDir) {
		return launch.Command{}, fmt.Errorf("mods folder %q is not an absolute path", req.ModsDir)
	}
	modsArgs := []string{"--mods-path", req.ModsDir}
	smapiArgs := appendLaunchArgs(goos, modsArgs, req.ExtraArgs)
	windows := goos == "windows"
	if req.Direct {
		if windows {
			return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, smapiMarker+".exe"), Args: smapiArgs}, nil
		}
		// The launcher reads its own flags before `--` and forwards only what follows it.
		return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, linuxLauncher), Args: smapiArgs}, nil
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
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{Name: steamPath, Args: append(appID, smapiArgs...), Failure: launch.HintSteam, Relay: true}, nil
}

func (g Game) vanillaCommand(goos string, req launch.Request, steamPath string) (launch.Command, error) {
	windows := goos == "windows"
	if req.Direct {
		name := "Stardew Valley.exe"
		if !windows {
			name = linuxOriginal
		}
		return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, name)}, nil
	}
	appID := []string{"-applaunch", g.SteamAppID()}
	switch {
	case req.Steam == nil:
		return launch.Command{}, launch.ErrNoSteam
	case windows:
		return launch.Command{Name: filepath.Join(req.Steam.Root, "steam.exe"), Args: appID, Failure: launch.HintSteam, Relay: true}, nil
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	return launch.Command{Name: steamPath, Args: appID, Failure: launch.HintSteam, Relay: true}, nil
}

// SteamLaunchForcesLoader reports whether Steam launch options run SMAPI in place of the game.
func (Game) SteamLaunchForcesLoader(options string) bool {
	return hasSMAPILine(options)
}

// hasSMAPILine reports whether Steam launch options run SMAPI in place of the game: `"<game>\StardewModdingAPI.exe" %command%`.
func hasSMAPILine(options string) bool {
	return strings.Contains(strings.ToLower(options), strings.ToLower(smapiMarker)) && strings.Contains(options, "%command%")
}
