package launchsvc

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GameToolPresent reports whether every file the catalog tool requires is in the profile's game install.
func (s *Service) GameToolPresent(gameID, profileID string, t components.GameTool) bool {
	inst, err := game.ResolveInstall(s.home, s.currentSettings(), gameID, s.profiles.InstallOf(gameID, profileID))
	if err != nil || inst.Dir == "" {
		return false
	}
	for _, rel := range t.Requires {
		if !safefs.IsFile(filepath.Join(inst.Dir, filepath.FromSlash(rel))) {
			return false
		}
	}
	return true
}

// RunGameTool runs a catalog tool of the game for the profile: the program under the install's own runtime, or the
// REDmod deploy on its own (Vortex's tools.redmodding.ts and tools.external.ts). A tool that Deploys, a launcher that
// starts the game, first places the profile's files and compiles its REDmods as a Play does, and leaves the files
// placed for the next Play to take back. The call returns once a deploy is done, and once a program has started.
func (s *Service) RunGameTool(ctx context.Context, gameID, profileID string, t components.GameTool) error {
	g, err := game.Require(gameID)
	if err != nil {
		return err
	}
	sl := s.profileSlot(g, profileID)
	if err := s.claimPreparing(g, keyOf(sl), profileID); err != nil {
		return err
	}
	defer s.donePreparing(sl)
	if ps, err := s.gameProcs(g); err == nil && len(ps) > 0 {
		return usererr.Wrap(usererr.Busy, fmt.Errorf("%s is already running", g.Name()))
	}
	inst, err := s.runnableInstall(g, launchTarget{profileID: profileID}, false)
	if err != nil {
		return err
	}
	if !s.GameToolPresent(gameID, profileID, t) {
		return usererr.New(usererr.NotFound, t.Name+" is not in the game's folder")
	}
	env := game.StartEnv{Direct: true, HideWindow: true, Ready: s.seen(g)}
	s.launchSettings(g, profileID, &env)
	s.directEnv(ctx, g, profileID, &env)
	s.locateSteam(&env, inst)
	var launchEnv map[string]string
	if t.Deploys || t.Action == components.ToolDeployREDmods {
		spec, err := s.profileSpec(ctx, gameID, profileID, "", false)
		if err != nil {
			return err
		}
		plan, err := s.planProfile(ctx, g, inst, profileID, launchModeOf(false), spec.Options, spec.Prefix, spec.Env)
		if err != nil {
			return err
		}
		dep, err := s.deployProfile(ctx, gameID, inst, profileID, plan)
		if err != nil {
			return err
		}
		if err := s.deployREDmods(ctx, gameID, inst, profileID, plan, env); err != nil {
			dep.unwind(ctx)
			return err
		}
		if t.Action == components.ToolDeployREDmods {
			dep.unwind(ctx)
			return nil
		}
		launchEnv = plan.Env
	}
	exe := filepath.Join(inst.Dir, filepath.FromSlash(t.Exe))
	cmd, err := game.ToolCommand(runtime.GOOS, inst, env, launchEnv, exe, t.Args)
	if err != nil {
		return err
	}
	exited, err := launch.StartWithEnv(context.WithoutCancel(ctx), cmd.Env, filepath.Dir(exe), cmd.Name, cmd.Args...)
	if err != nil {
		return err
	}
	go launch.ReleaseOnExit(exited)
	return nil
}
