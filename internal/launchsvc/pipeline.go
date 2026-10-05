package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// loaderID is the id of the game's first loader, "" when it has none.
func loaderID(gameID string) string {
	if l, ok := game.PrimaryLoader(gameID); ok {
		return l.ID()
	}
	return ""
}

// view is the profile as loaders see it.
func (s *Service) view(g game.Game, inst game.Install, profileID string) (loader.ProfileView, error) {
	dir, err := s.profiles.ProfileDir(g.ID(), profileID)
	if err != nil {
		return loader.ProfileView{}, err
	}
	companion, _ := s.bridgeFolder(g, profileID)
	return loader.ProfileView{Game: g.ID(), Dir: dir, InstallDir: inst.Dir, Runtime: inst.Runtime, Companion: companion}, nil
}

// launchPlan is what a launch starts. Loaders contribute first; the profile's own launch options, prefix and
// environment follow, so its options come after the loader's arguments. A vanilla plan has the loader's way of
// starting the game without it and nothing of the profile's.
func (s *Service) launchPlan(ctx context.Context, g game.Game, inst game.Install, profileID string, mode launchplan.Mode, options, prefix, env string) (*launchplan.Plan, error) {
	l, ok := game.PrimaryLoader(g.ID())
	if !ok {
		return nil, fmt.Errorf("%s has no loader", g.Name())
	}
	plan := launchplan.New(mode)
	if mode == launchplan.ModeVanilla {
		if v, ok := l.(loader.Vanilla); ok {
			return plan, v.Vanilla(ctx, plan, loader.Target{Game: g.ID(), InstallDir: inst.Dir, Runtime: inst.Runtime})
		}
		return plan, nil
	}
	view, err := s.view(g, inst, profileID)
	if err != nil {
		return nil, err
	}
	if err := l.Contribute(ctx, plan, view); err != nil {
		return nil, err
	}
	extra, err := game.ParseLaunchOptions(g.ID(), options)
	if err != nil {
		return nil, err
	}
	plan.AddArgs(extra...)
	if plan.Prefix, err = profile.LaunchPrefixArgs(prefix); err != nil {
		return nil, err
	}
	pairs, err := profile.LaunchEnvironment(env)
	if err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		k, v, _ := strings.Cut(pair, "=")
		plan.SetEnv(k, v)
	}
	return plan, nil
}

// deployerID is the deployer that places a plan's install-side files into the game folder. A loader that redirects
// the game to the profile's folder (SMAPI's --mods-path) declares no such files, so its launches skip the deploy.
const deployerID = "link-into-install"

// deployment is the placed files of one launch, taken back once the game has exited.
type deployment struct {
	d deploy.Deployer
	m deploy.Manifest
	// finish unwinds under a context that outlives the launch call, for the waiter, which has none of its own.
	finish func()
	once   sync.Once
}

// journalDir is where an install's deploy journal lives: beside Mortar's data, never inside the install, so a game
// reinstall cannot take the record of what was placed with it.
func journalDir(installID string) (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "journal", installID), nil
}

// deployProfile is startDeploy for a launch of profileID: a game that deploys into its install also places the
// profile's packages.
func (s *Service) deployProfile(ctx context.Context, gameID string, inst game.Install, profileID string, plan *launchplan.Plan) (*deployment, error) {
	var in profile.DeployInputs
	if plan.Mode != launchplan.ModeVanilla && profileID != "" && s.profiles != nil {
		var err error
		if in, err = s.profiles.DeployInputs(gameID, profileID, inst.Dir); err != nil {
			return nil, err
		}
	}
	return startDeploy(ctx, inst, plan, in)
}

// startDeploy places plan's install-side files and the profile's packages into inst, journaled before the first file
// moves. A launch with neither returns a deployment that has nothing to take back.
func startDeploy(ctx context.Context, inst game.Install, plan *launchplan.Plan, in profile.DeployInputs) (*deployment, error) {
	if plan.Mode == launchplan.ModeVanilla || (len(plan.Files) == 0 && len(in.Packages) == 0) {
		return &deployment{}, nil
	}
	d, ok := deploy.Get(deployerID)
	if !ok {
		return nil, fmt.Errorf("no deployer %q", deployerID)
	}
	dir, err := journalDir(inst.ID)
	if err != nil {
		return nil, err
	}
	p, err := d.Plan(deploy.View{JournalDir: dir, Overwrite: in.Overwrite}, deploy.InstallView{Dir: inst.Dir, Targets: in.Targets}, in.Packages, plan.Files)
	if err != nil {
		return nil, err
	}
	m, err := d.Apply(ctx, p)
	dep := &deployment{d: d, m: m}
	detached := context.WithoutCancel(ctx)
	dep.finish = func() { dep.unwind(detached) }
	if err != nil {
		if len(m.Ops) > 0 {
			dep.unwind(context.WithoutCancel(ctx))
		}
		return nil, err
	}
	return dep, nil
}

// unwind takes back what the game wrote, then removes the placed files. It is safe to call again, on nil and on a
// deployment that placed nothing.
func (dep *deployment) unwind(ctx context.Context) {
	if dep == nil || dep.d == nil {
		return
	}
	dep.once.Do(func() {
		if _, err := dep.d.Harvest(ctx, dep.m); err != nil {
			log.Printf("launch: harvest: %v", err)
		}
		if err := dep.d.Purge(ctx, dep.m); err != nil {
			log.Printf("launch: purge: %v", err)
		}
	})
}

// RecoverDeploys finishes the deploy of any launch Mortar did not see end (a crash or a quit while the game ran), for
// every install whose game is no longer running.
//
//wails:ignore
func (s *Service) RecoverDeploys(ctx context.Context) error {
	d, ok := deploy.Get(deployerID)
	if !ok {
		return nil
	}
	var errs []error
	for _, id := range game.Implemented() {
		g := game.Find(id)
		for _, sl := range s.slots(g) {
			if sl.inst == "" {
				continue
			}
			dir, err := journalDir(sl.inst)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			alive := func() bool {
				procs, err := s.gameProcs(sl)
				return err != nil || len(procs) > 0
			}
			if err := d.Recover(ctx, dir, alive); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", id, err))
			}
		}
	}
	return errors.Join(errs...)
}

// ensureRuntime makes the runtime provide what the plan asks for before the game starts. A Wine prefix that does not
// exist yet is left to Steam, which creates it on the first run; the doctor reports it until then. These edits are
// persistent and idempotent, so they are not journaled.
func (s *Service) ensureRuntime(inst game.Install, reqs []launchplan.RuntimeReq) error {
	for _, r := range reqs {
		if r.Kind != "dll-override" || r.Key != "winhttp" {
			continue
		}
		compat, ok := inst.CompatData(s.home)
		if !ok {
			continue
		}
		reg := filepath.Join(compat, "pfx", "user.reg")
		if _, err := os.Stat(reg); err != nil {
			continue
		}
		if err := bepinex5.EnsureWinHTTPOverride(reg); err != nil {
			return fmt.Errorf("make the Proton prefix load winhttp: %w", err)
		}
	}
	return nil
}
