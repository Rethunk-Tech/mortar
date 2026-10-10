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

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/iniedit"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	gameruntime "github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/savesiso"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// loaderID is the id of the game's first loader, "" when it has none.
func loaderID(gameID string) string {
	if l, ok := game.PrimaryLoader(gameID); ok {
		return l.ID()
	}
	return ""
}

// profileLoader is the id of the loader the profile runs; "" (the game's primary loader) without a profile.
func (s *Service) profileLoader(gameID, profileID string) string {
	if s.profiles == nil || profileID == "" {
		return ""
	}
	return s.profiles.LoaderID(gameID, profileID)
}

// loaderOf is the loader the profile runs.
func (s *Service) loaderOf(gameID, profileID string) (loader.Loader, bool) {
	return game.LoaderOf(gameID, s.profileLoader(gameID, profileID))
}

// view is the profile as loaders see it.
func (s *Service) view(g game.Game, inst game.Install, profileID string) (loader.ProfileView, error) {
	dir, err := s.profiles.ProfileDir(g.ID(), profileID)
	if err != nil {
		return loader.ProfileView{}, err
	}
	companion, _ := s.bridgeFolder(g, profileID)
	return loader.ProfileView{Game: g.ID(), Dir: dir, InstallDir: inst.Dir, Runtime: inst.Runtime, Platform: inst.Platform, Companion: companion}, nil
}

// launchPlan is what a launch starts. Loaders contribute first; the profile's own launch options, prefix and
// environment follow, so its options come after the loader's arguments. A vanilla plan has the loader's way of
// starting the game without it and nothing of the profile's.
func (s *Service) launchPlan(ctx context.Context, g game.Game, inst game.Install, profileID string, mode launchplan.Mode, options, prefix, env string) (*launchplan.Plan, error) {
	l, ok := s.loaderOf(g.ID(), profileID)
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
	if err := s.armIntroSkip(g, l, profileID, view.Dir); err != nil {
		return nil, err
	}
	plan.AddArgs(s.graphicsArgs(g, profileID)...)
	extra, err := game.ParseLaunchOptions(g.ID(), s.profileLoader(g.ID(), profileID), options)
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
	if inst.Runtime == gameruntime.Proton || inst.Runtime == gameruntime.WinePrefix {
		plan.ApplyDLLOverrides()
	}
	return plan, nil
}

// deployerID is the deployer that places a plan's install-side files into the game folder. A loader that redirects
// the game to the profile's folder (SMAPI's --mods-path) declares no such files, so its launches skip the deploy.
const deployerID = "copy-into-install"

// deployment is the placed files of one launch, taken back once the game has exited.
type deployment struct {
	d deploy.Deployer
	m deploy.Manifest
	// finish unwinds under a context that outlives the launch call, for the waiter, which has none of its own.
	finish func()
	once   sync.Once
	// saves is the swap that gives the profile its own saves folder, undone with the placed files.
	saves *savesiso.Manifest
	// options is the swap that gives the profile its own copy of the game's options file.
	options *savesiso.FileManifest
}

// savesJournal is the swap's journal folder, a sibling of the deploy journal so neither takes the other's record.
func savesJournal(installID string) (string, error) { return journalDir(installID + "-saves") }

// pathOptions is the catalog path role of a game's options file, a single file the game rewrites in place.
const pathOptions = "options"

// optionsJournal is the options swap's journal folder, beside the saves journal so neither takes the other's record.
func optionsJournal(installID string) (string, error) { return journalDir(installID + "-options") }

// swapOptions puts the profile's own copy of the game's options file in place for this launch and sets the player's
// file aside; the swap is undone with dep. The profile's copy starts as the player's file.
func (s *Service) swapOptions(ctx context.Context, gameID string, inst game.Install, profileID, installID string, dep *deployment) error {
	if !game.HasPath(gameID, pathOptions) {
		return nil
	}
	target, err := game.PathFor(s.home, s.settings.Get(), gameID, s.pinOf(gameID, profileID, installID), pathOptions)
	if err != nil {
		return err
	}
	profileDir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return err
	}
	own := filepath.Join(profileDir, filepath.Base(target))
	dir, err := optionsJournal(inst.ID)
	if err != nil {
		return err
	}
	if err := savesiso.RecoverFile(dir, nil); err != nil {
		return err
	}
	if err := savesiso.SeedFile(target, own); err != nil {
		return err
	}
	if s.gameSettingsMode(gameID, profileID, inst.ID) == settings.GameSettingsEdit {
		if err := s.applyRequiredSettings(gameID, own); err != nil {
			return err
		}
	}
	m, err := savesiso.ApplyFile(dir, target, own)
	if err != nil {
		return err
	}
	dep.options = &m
	if dep.finish == nil {
		detached := context.WithoutCancel(ctx)
		dep.finish = func() { dep.unwind(detached) }
	}
	return nil
}

// gameSettingsMode is the profile's gameSettingsMode: edit makes the options file hold the catalog's required settings.
func (s *Service) gameSettingsMode(gameID, profileID, installID string) string {
	return settings.ResolveAt(s.settings.Get(), "gameSettingsMode", settings.Scope{Game: gameID, Install: s.pinOf(gameID, profileID, installID), Profile: profileID}, launchOverrides(s.profiles, gameID, profileID))
}

// applyRequiredSettings writes the catalog's required settings for the options role into the profile's copy. A profile
// without a copy (the player had no file yet) is left for the game to create.
func (s *Service) applyRequiredSettings(gameID, own string) error {
	info, ok := components.Game(gameID)
	if !ok || !fsx.IsFile(own) {
		return nil
	}
	b, err := fsx.ReadFile(own)
	if err != nil {
		return err
	}
	text := string(b)
	for _, r := range info.RequiredSettings {
		if r.Path == pathOptions {
			if text, err = iniedit.Set(text, "", r.Key, r.Value); err != nil {
				return fmt.Errorf("%s: %w", own, err)
			}
		}
	}
	if text == string(b) {
		return nil
	}
	return datadir.WriteFile(own, []byte(text), 0o600)
}

// swapSaves points the game's save folder at the profile's own for this launch when the profile keeps its saves
// separate; the swap is undone with dep.
func (s *Service) swapSaves(ctx context.Context, gameID string, inst game.Install, profileID, installID string, dep *deployment) error {
	if !game.HasSaves(gameID) || !s.profiles.SeparateSaves(gameID, profileID) {
		return nil
	}
	own, err := s.profiles.SavesFolder(gameID, profileID)
	if err != nil {
		return err
	}
	shared, err := game.SavesDir(s.home, s.settings.Get(), gameID, s.pinOf(gameID, profileID, installID))
	if err != nil {
		return err
	}
	dir, err := savesJournal(inst.ID)
	if err != nil {
		return err
	}
	if err := savesiso.Recover(dir, nil); err != nil {
		return err
	}
	m, err := savesiso.Apply(dir, shared, own, false)
	if err != nil {
		return err
	}
	dep.saves = &m
	if dep.finish == nil {
		detached := context.WithoutCancel(ctx)
		dep.finish = func() { dep.unwind(detached) }
	}
	return nil
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

// deployProfile is startDeploy for a launch of profileID: a game whose profile holds the loader first lays the
// profile's packages out in the profile, then places the loader's install-side files.
func (s *Service) deployProfile(ctx context.Context, gameID string, inst game.Install, profileID string, plan *launchplan.Plan) (*deployment, error) {
	if plan.Mode != launchplan.ModeVanilla && profileID != "" && s.profiles != nil {
		if err := s.profiles.SyncPackages(gameID, profileID); err != nil {
			return nil, err
		}
		if err := s.prelaunch(gameID, profileID, inst.Dir); err != nil {
			return nil, err
		}
	}
	roots, err := s.planRoots(gameID, inst, plan)
	if err != nil {
		return nil, err
	}
	return startDeploy(ctx, inst, plan, roots)
}

// planRoots resolves the catalog path roles the plan's files are placed under (a mods folder outside the install).
func (s *Service) planRoots(gameID string, inst game.Install, plan *launchplan.Plan) (map[string]string, error) {
	var roots map[string]string
	for _, f := range plan.Files {
		if f.Root == "" || roots[f.Root] != "" {
			continue
		}
		dir, err := game.PathFor(s.home, s.settings.Get(), gameID, inst.ID, f.Root)
		if err != nil {
			return nil, err
		}
		if roots == nil {
			roots = map[string]string{}
		}
		roots[f.Root] = dir
	}
	return roots, nil
}

func (s *Service) prelaunch(gameID, profileID, installDir string) error {
	l, ok := s.loaderOf(gameID, profileID)
	pre, isPre := l.(loader.Prelaunch)
	if !ok || !isPre {
		return nil
	}
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return err
	}
	return pre.Prelaunch(loader.ProfileView{Game: gameID, Dir: dir, InstallDir: installDir})
}

// startDeploy places plan's install-side files into inst, journaled before the first file moves. A launch with none
// returns a deployment that has nothing to take back.
func startDeploy(ctx context.Context, inst game.Install, plan *launchplan.Plan, roots map[string]string) (*deployment, error) {
	if plan.Mode == launchplan.ModeVanilla || len(plan.Files) == 0 {
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
	// begin refused a running game, so a journal still here belongs to a launch nothing is taking back.
	if err := d.Recover(ctx, dir, nil); err != nil {
		return nil, err
	}
	p, err := d.Plan(deploy.View{JournalDir: dir, Roots: roots}, inst.Dir, plan.Files)
	if err != nil {
		return nil, err
	}
	m, err := d.Apply(ctx, p)
	dep := &deployment{d: d, m: m}
	detached := context.WithoutCancel(ctx)
	dep.finish = func() { dep.unwind(detached) }
	if err != nil {
		if len(m.Ops) > 0 {
			dep.unwind(detached)
		}
		return nil, err
	}
	return dep, nil
}

// unwind removes the placed files. It is safe to call again, on nil and on a deployment that placed nothing.
func (dep *deployment) unwind(ctx context.Context) {
	if dep == nil {
		return
	}
	dep.once.Do(func() {
		if dep.d != nil {
			if err := dep.d.Purge(ctx, dep.m); err != nil {
				log.Printf("launch: purge: %v", err)
			}
		}
		if dep.saves != nil {
			if err := savesiso.Purge(*dep.saves); err != nil {
				log.Printf("launch: restore saves: %v", err)
			}
		}
		if dep.options != nil {
			if err := savesiso.PurgeFile(*dep.options); err != nil {
				log.Printf("launch: restore options file: %v", err)
			}
		}
	})
}

// RecoverDeploys finishes the deploy of any launch Mortar did not see end (a crash or a quit while the game ran), for
// every install whose game is no longer running, and returns the games it found such a journal for.
//
//wails:ignore
func (s *Service) RecoverDeploys(ctx context.Context) ([]string, error) {
	var errs []error
	var found []string
	for _, id := range game.Implemented() {
		ok, err := s.RecoverGameDeploys(ctx, id)
		if ok {
			found = append(found, id)
		}
		errs = append(errs, err)
	}
	return found, errors.Join(errs...)
}

// RecoverGameDeploys is RecoverDeploys for one game; found reports whether it had a journal.
//
//wails:ignore
func (s *Service) RecoverGameDeploys(ctx context.Context, id string) (found bool, err error) {
	d, ok := deploy.Get(deployerID)
	g := game.Find(id)
	if !ok || g == nil {
		return false, nil
	}
	var errs []error
	for _, sl := range s.slots(g) {
		if sl.inst == "" {
			continue
		}
		dir, err := journalDir(sl.inst)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		alive := func() bool { return s.slotInUse(sl) }
		if sj, err := savesJournal(sl.inst); deploy.HasJournal(dir) || (err == nil && savesiso.HasJournal(sj)) {
			found = true
		}
		if oj, err := optionsJournal(sl.inst); err == nil && savesiso.HasFileJournal(oj) {
			found = true
		}
		if err := d.Recover(ctx, dir, alive); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
		if sj, err := savesJournal(sl.inst); err != nil {
			errs = append(errs, err)
		} else if err := savesiso.Recover(sj, alive); err != nil {
			errs = append(errs, fmt.Errorf("%s saves: %w", id, err))
		}
		if oj, err := optionsJournal(sl.inst); err != nil {
			errs = append(errs, err)
		} else if err := savesiso.RecoverFile(oj, alive); err != nil {
			errs = append(errs, fmt.Errorf("%s options file: %w", id, err))
		}
	}
	return found, errors.Join(errs...)
}

// slotInUse reports whether the install's game may be using its deploy and saves: a process runs, or a launch is
// still being prepared or started, whose journal is its own and not a crash's. A failed process check counts as in use.
func (s *Service) slotInUse(sl slot) bool {
	if procs, err := s.gameProcs(sl); err != nil || len(procs) > 0 {
		return true
	}
	s.mu.Lock()
	_, preparing := s.preparing[keyOf(sl)]
	s.mu.Unlock()
	return preparing || s.statusOf(sl).State.Active()
}

// LeftoverJournals lists the deploy and save swap journals of the game's installs that RecoverDeploys would finish:
// those of installs whose game is not running.
//
//wails:ignore
func (s *Service) LeftoverJournals(gameID string) []string {
	g := game.Find(gameID)
	if g == nil {
		return nil
	}
	var out []string
	for _, sl := range s.slots(g) {
		if sl.inst == "" {
			continue
		}
		if s.slotInUse(sl) {
			continue
		}
		if dir, err := journalDir(sl.inst); err == nil && deploy.HasJournal(dir) {
			out = append(out, dir)
		}
		if dir, err := savesJournal(sl.inst); err == nil && savesiso.HasJournal(dir) {
			out = append(out, dir)
		}
		if dir, err := optionsJournal(sl.inst); err == nil && savesiso.HasFileJournal(dir) {
			out = append(out, dir)
		}
	}
	return out
}

// errNoPrefix is a Steam-relayed Proton launch whose prefix does not exist yet: Steam would create it without the
// loader's DLL override, and nothing Mortar passes reaches the game.
var errNoPrefix = errors.New("the Proton prefix does not exist yet")

// ensureRuntime makes the runtime provide what the plan asks for before the game starts. A direct launch passes the
// override in the environment (ApplyDLLOverrides), so a prefix Wine has not made yet is harmless; a launch Steam
// relays reads the prefix's registry, which must exist to carry the override. These edits are persistent and
// idempotent, so they are not journaled.
func (s *Service) ensureRuntime(inst game.Install, reqs []launchplan.RuntimeReq, direct bool) error {
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
			if direct {
				continue
			}
			return errNoPrefix
		}
		if err := bepinex5.EnsureWinHTTPOverride(reg); err != nil {
			return fmt.Errorf("make the Proton prefix load winhttp: %w", err)
		}
	}
	return nil
}
