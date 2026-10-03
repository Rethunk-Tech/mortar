package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

const (
	bisectRunTimeout   = 3 * time.Minute
	bisectStartupGrace = 3 * time.Second
)

// RunForBisect launches one profile and returns whether it reached a healthy running state.
func (s *Service) RunForBisect(ctx context.Context, gameID, profileID string, direct bool) (bool, launch.Summary, error) {
	runCtx, cancel := context.WithTimeout(ctx, bisectRunTimeout)
	defer cancel()
	beforeRunID, _, err := s.LastRunSummary(gameID, profileID)
	if err != nil {
		return false, launch.Summary{}, err
	}
	if err := s.start(runCtx, gameID, profileID, direct); err != nil {
		return false, launch.Summary{}, err
	}

	started := false
	var runningSince time.Time
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
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
			if runningSince.IsZero() {
				runningSince = time.Now()
			}
			// SMAPI reports skipped startup mods after the process is Running, so allow its
			// initial log batch to arrive before treating Running as a healthy title screen.
			if time.Since(runningSince) >= bisectStartupGrace {
				if err := s.Stop(gameID); err != nil {
					return false, launch.Summary{}, err
				}
				return true, launch.Summary{}, nil
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

// bisectStartupFailure reports a SMAPI crash report while the game runs. Plain error lines (a skipped mod, missing
// Steam) do not fail a bisect step: the bisect looks for the mod behind a crash, and counting errors would blame
// whichever mod logs one.
func bisectStartupFailure(lines []launch.Entry) bool {
	return slices.ContainsFunc(lines, func(entry launch.Entry) bool { return entry.Level == launch.Alert })
}

func summaryHealthy(summary launch.Summary) bool {
	return !summary.Crashed
}

func (s *Service) stopBisectRun(gameID string) error {
	status, err := s.Status(gameID)
	if err != nil || (status.State != Running && status.State != Launching) {
		return err
	}
	if status.State == Running {
		return s.Stop(gameID)
	}
	return nil
}
