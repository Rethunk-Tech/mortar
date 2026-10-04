package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/bridge"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

const bisectRunTimeout = 3 * time.Minute

// Mods commonly crash on SMAPI's GameLaunched, which fires once the title screen has loaded: several seconds
// after the process counts as Running. A shorter grace calls the crashing step healthy and blames the other half.
// Used only when the profile has no bridge folder to write a title-screen report.
var bisectStartupGrace = 20 * time.Second

// RunForBisect launches one profile the way Play would (the profile's launch method) and returns whether it reached a
// healthy running state. A new startup report after launch means the title screen was reached; otherwise the step
// waits for a crash, exit, the 20 s grace when the bridge is absent, or bisectRunTimeout.
func (s *Service) RunForBisect(ctx context.Context, gameID, profileID string) (bool, launch.Summary, error) {
	runCtx, cancel := context.WithTimeout(ctx, bisectRunTimeout)
	defer cancel()
	beforeRunID, _, err := s.LastRunSummary(gameID, profileID)
	if err != nil {
		return false, launch.Summary{}, err
	}
	launched := time.Now()
	if err := s.start(runCtx, gameID, profileID, s.launchesDirect(gameID, profileID)); err != nil {
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

	started := false
	var runningSince time.Time
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if hasBridge && startupReportAfter(startup, launched) {
			s.waitSampled(gameID, time.Minute)
			if err := s.Stop(gameID); err != nil {
				return false, launch.Summary{}, err
			}
			return true, launch.Summary{}, nil
		}
		status, err := s.Status(gameID)
		if err != nil {
			return false, launch.Summary{}, err
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
				if err := s.Stop(gameID); err != nil {
					return false, launch.Summary{}, err
				}
				return false, launch.Summary{}, nil
			}
			if !hasBridge {
				if runningSince.IsZero() {
					runningSince = time.Now()
				}
				if time.Since(runningSince) >= bisectStartupGrace {
					if err := s.Stop(gameID); err != nil {
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
			_ = s.stopBisectRun(gameID)
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
	matches, err := filepath.Glob(filepath.Join(modsDir, "*", bridge.ModFolder))
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

func (s *Service) launchesDirect(gameID, profileID string) bool {
	method := settings.Resolve(s.settings.Get(), "defaultLaunchMethod", gameID, launchOverrides(s.profiles, gameID, profileID))
	return method == settings.LaunchDirect
}

func (s *Service) stopBisectRun(gameID string) error {
	status, err := s.Status(gameID)
	if err != nil || !status.State.Active() {
		return err
	}
	if status.State == Running {
		return s.Stop(gameID)
	}
	return nil
}
