package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const bisectRunTimeout = 3 * time.Minute

// Mods commonly crash on SMAPI's GameLaunched, which fires once the title screen has loaded: several seconds
// after the process counts as Running. A shorter grace calls the crashing step healthy and blames the other half.
// Used only when the profile has no bridge folder to write a title-screen report.
var bisectStartupGrace = 20 * time.Second

// bisectSceneSettle is how long a BepInEx game must hold a scene, after leaving its first, to count as healthy: Lethal
// Company's mods most often crash as a scene loads, so the step waits past the load rather than for the chainloader.
var bisectSceneSettle = 10 * time.Second

// bridgeScene is the Mortar BepInEx Bridge's line for each scene the game loads.
var bridgeScene = regexp.MustCompile(`^Bridge plugin alive after scene (.+): \w+$`)

// sceneWatch follows a BepInEx run's log to the point it counts as healthy.
type sceneWatch struct {
	scene        string
	scenes       int
	since, ready time.Time
}

// settled reports a BepInEx run healthy once its game has held a scene the bridge reported for bisectSceneSettle, or for
// bisectStartupGrace while that is still the first scene (Lethal Company waits there for the player's Online or LAN
// choice). Without the bridge it takes the chainloader finishing plus bisectStartupGrace. A log that shows neither
// is not yet healthy: a slow start under Proton is not a pass.
func (w *sceneWatch) settled(lines []launch.Entry, now time.Time) bool {
	scene, ready := "", false
	for _, e := range lines {
		if m := bridgeScene.FindStringSubmatch(e.Message); m != nil {
			scene = m[1]
		}
		ready = ready || bepinex5.Loader{}.Ready(e.Message)
	}
	if scene != "" {
		if scene != w.scene {
			w.scene, w.since = scene, now
			w.scenes++
		}
		hold := bisectSceneSettle
		if w.scenes == 1 {
			hold = bisectStartupGrace
		}
		return now.Sub(w.since) >= hold
	}
	if ready && w.ready.IsZero() {
		w.ready = now
	}
	return ready && now.Sub(w.ready) >= bisectStartupGrace
}

// RunForBisect launches one profile the way Play would (the profile's launch method) and returns whether it reached a
// healthy running state. A new startup report after launch means the title screen was reached, and a BepInEx game is
// healthy once it settles in a scene (see sceneWatch); otherwise the step waits for a crash, exit, the 20 s grace when
// the bridge is absent, or bisectRunTimeout.
//
//wails:ignore
func (s *Service) RunForBisect(ctx context.Context, gameID, profileID string) (bool, launch.Summary, error) {
	return s.runForInstall(ctx, gameID, profileID, "")
}

// runForInstall is RunForBisect on the install with this id ("" is the profile's own).
func (s *Service) runForInstall(ctx context.Context, gameID, profileID, installID string) (bool, launch.Summary, error) {
	runCtx, cancel := context.WithTimeout(ctx, bisectRunTimeout)
	defer cancel()
	g, err := game.Require(gameID)
	if err != nil {
		return false, launch.Summary{}, err
	}
	sl := s.profileSlot(g, profileID)
	if installID != "" {
		if sl, err = s.installSlot(gameID, installID); err != nil {
			return false, launch.Summary{}, err
		}
	}
	beforeRunID, _, err := s.LastRunSummary(gameID, profileID)
	if err != nil {
		return false, launch.Summary{}, err
	}
	launched := time.Now()
	if err := s.start(runCtx, gameID, profileID, installID, "", s.LaunchesDirect(gameID, profileID)); err != nil {
		return false, launch.Summary{}, err
	}

	startup := ""
	modsDir := ""
	if s.profiles != nil {
		if dir, err := s.profiles.ProfileDir(gameID, profileID); err == nil {
			startup = filepath.Join(dir, startupDir)
		}
		if dir, err := s.profiles.ModsDir(gameID, profileID); err == nil {
			modsDir = dir
		}
	}
	hasBridge := profileHasBridge(modsDir)
	var scenes *sceneWatch
	if s.profileLoader(gameID, profileID) == bepinex5.ID {
		scenes = &sceneWatch{}
	}

	started := false
	var runningSince time.Time
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if hasBridge && startupReportAfter(startup, launched) {
			s.waitSampled(sl, time.Minute)
			if err := s.stopSlot(runCtx, sl); err != nil {
				return false, launch.Summary{}, err
			}
			return true, launch.Summary{}, nil
		}
		status := s.statusOf(sl)
		if status.State == Idle {
			if why := s.startFailure(sl); why != "" {
				return false, launch.Summary{}, fmt.Errorf("the game did not start: %s", why)
			}
		}
		switch status.State {
		case Launching:
			started = true
		case Running:
			started = true
			lines, err := s.Lines(gameID, profileID)
			if err != nil {
				return false, launch.Summary{}, err
			}
			if bisectStartupFailure(lines) {
				if err := s.stopSlot(runCtx, sl); err != nil {
					return false, launch.Summary{}, err
				}
				return false, launch.Summary{}, nil
			}
			if scenes != nil {
				if scenes.settled(lines, time.Now()) {
					if err := s.stopSlot(runCtx, sl); err != nil {
						return false, launch.Summary{}, err
					}
					return true, launch.Summary{}, nil
				}
			} else if !hasBridge {
				if runningSince.IsZero() {
					runningSince = time.Now()
				}
				if time.Since(runningSince) >= bisectStartupGrace {
					if err := s.stopSlot(runCtx, sl); err != nil {
						return false, launch.Summary{}, err
					}
					return true, launch.Summary{}, nil
				}
			}
		case NoSteam:
			return false, launch.Summary{}, errors.New("the crash check needs a direct launch")
		case Failed:
			if !started {
				return false, launch.Summary{}, fmt.Errorf("bisect launch failed: %s", status.Error)
			}
			runID, summary, err := s.LastRunSummary(gameID, profileID)
			if runID == beforeRunID {
				return false, summary, err
			}
			return summaryHealthy(summary), summary, err
		case Idle:
			runID, summary, err := s.LastRunSummary(gameID, profileID)
			if (started || runID != "") && runID != beforeRunID {
				return summaryHealthy(summary), summary, err
			}
		}

		select {
		case <-runCtx.Done():
			_ = s.stopBisectRun(context.WithoutCancel(runCtx), sl)
			if errors.Is(ctx.Err(), context.Canceled) {
				return false, launch.Summary{}, ctx.Err()
			}
			return false, launch.Summary{}, nil
		case <-ticker.C:
		}
	}
}

func profileHasBridge(modsDir string) bool {
	if modsDir == "" {
		return false
	}
	// The bridge sits inside its store entry's folder, like every installed mod.
	matches, err := filepath.Glob(filepath.Join(modsDir, "*", bridge.SMAPI.ModFolder))
	return err == nil && len(matches) > 0
}

func startupReportAfter(dir string, since time.Time) bool {
	if dir == "" {
		return false
	}
	names, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return false
	}
	for _, name := range names {
		info, err := os.Stat(name)
		if err != nil {
			continue
		}
		if info.ModTime().After(since) {
			return true
		}
	}
	return false
}

// bisectStartupFailure reports a game crash while the game runs. Mod errors, including a mod that crashed on entry,
// do not fail a bisect step: the bisect looks for the mod behind a crash, and counting errors would blame whichever
// mod logs one.
func bisectStartupFailure(lines []launch.Entry) bool {
	return slices.ContainsFunc(lines, launch.IsCrash)
}

func summaryHealthy(summary launch.Summary) bool {
	return !summary.Crashed
}

// LaunchesDirect reports whether the profile's launch method (or the game's, for profileID "") skips Steam.
//
//wails:ignore
func (s *Service) LaunchesDirect(gameID, profileID string) bool {
	method := settings.ResolveAt(s.settings.Get(), "defaultLaunchMethod", settings.Scope{Game: gameID, Install: s.profiles.InstallOf(gameID, profileID), Profile: profileID}, launchOverrides(s.profiles, gameID, profileID))
	return method == settings.LaunchDirect
}

func (s *Service) stopBisectRun(ctx context.Context, sl slot) error {
	if status := s.statusOf(sl); status.State == Running {
		return s.stopSlot(ctx, sl)
	}
	return nil
}
