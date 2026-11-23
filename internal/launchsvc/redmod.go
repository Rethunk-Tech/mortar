package launchsvc

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Rethunk-Tech/go-bepinex/wineoverride"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/redmod"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// deployREDmods compiles the profile's REDmods with the game's own REDmod tool once they are placed in the install,
// and starts the game with -modded so it loads them (Vortex's tools.redmodding.ts). An install without the REDmod DLC
// starts as it is; a deploy the tool reports as failed stops the launch with the tool's reason.
func (s *Service) deployREDmods(ctx context.Context, gameID string, inst game.Install, profileID string, plan *launchplan.Plan, env game.StartEnv) error {
	info, _ := components.Game(gameID)
	if t, ok := info.Target(installer.TargetMods); !ok || t.Layout != components.LayoutCyberpunk {
		return nil
	}
	names, err := s.profiles.REDmodFolders(gameID, profileID)
	if err != nil || len(names) == 0 {
		return err
	}
	if !redmod.Installed(inst.Dir) {
		slog.Info("REDmod is not installed, launching without deploying", "game", gameID, "dir", inst.Dir)
		return nil
	}
	path := func(p string) string { return p }
	if runtime.GOOS != "windows" {
		path = wineoverride.ZPath
	}
	err = redmod.Deploy(ctx, inst.Dir, names, path, func(ctx context.Context, exe string, args []string) (int, string, error) {
		cmd, err := game.ToolCommand(runtime.GOOS, inst, env, plan.Env, exe, args)
		if err != nil {
			return 0, "", err
		}
		return launch.RunTool(ctx, cmd.Env, filepath.Dir(exe), cmd.Name, cmd.Args...)
	})
	if err != nil {
		return usererr.Wrap(usererr.Invalid, err)
	}
	if !slices.Contains(plan.Args, redmod.Arg) {
		plan.AddArgs(redmod.Arg)
	}
	return nil
}
