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

// Launch starts SMAPI on req.ModsDir and returns once SMAPI has rewritten its log, sending the log's lines to
// onLine. It returns launch.ErrNoSteam when there is no Steam to launch through and req.Direct is false.
func (g Game) Launch(ctx context.Context, req launch.Request, onLine func(string)) error {
	logDir, err := g.logDir()
	if err != nil {
		return err
	}
	lookPath := g.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	steamPath, _ := lookPath("steam")
	cmd, err := g.command(runtime.GOOS, req, steamPath)
	if err != nil {
		return err
	}
	cmd.LogFile = filepath.Join(logDir, "SMAPI-latest.txt")
	run := g.Runner
	if run == nil {
		run = launch.Start
	}
	return launch.Run(ctx, run, cmd, g.LaunchTiming, onLine)
}

// command builds the process to start. steamPath is the `steam` found on PATH, "" when there is none.
func (g Game) command(goos string, req launch.Request, steamPath string) (launch.Command, error) {
	if !filepath.IsAbs(req.ModsDir) {
		return launch.Command{}, fmt.Errorf("mods folder %q is not an absolute path", req.ModsDir)
	}
	modsArgs := []string{"--mods-path", req.ModsDir}
	windows := goos == "windows"
	if req.Direct {
		if windows {
			return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, smapiMarker+".exe"), Args: modsArgs}, nil
		}
		// The launcher reads its own flags before `--` and forwards only what follows it.
		args := append([]string{"--skip-terminal", "--"}, modsArgs...)
		return launch.Command{Dir: req.InstallDir, Name: filepath.Join(req.InstallDir, linuxLauncher), Args: args}, nil
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
		return launch.Command{Name: filepath.Join(req.Steam.Root, "steam.exe"), Args: append(appID, modsArgs...), Failure: hint}, nil
	case steamPath == "":
		return launch.Command{}, launch.ErrNoSteam
	}
	args := append(append(appID, "--skip-terminal", "--"), modsArgs...)
	return launch.Command{Name: steamPath, Args: args, Failure: launch.HintSteam}, nil
}

// hasSMAPILine reports whether Steam launch options run SMAPI in place of the game: `"<game>\StardewModdingAPI.exe" %command%`.
func hasSMAPILine(options string) bool {
	return strings.Contains(strings.ToLower(options), strings.ToLower(smapiMarker)) && strings.Contains(options, "%command%")
}
