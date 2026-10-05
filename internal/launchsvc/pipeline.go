package launchsvc

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/game/stardew"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

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
	extra, err := stardew.ParseLaunchOptions(options)
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
