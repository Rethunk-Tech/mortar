package launchsvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

const bisectRunTimeout = 3 * time.Minute

// RunForBisect launches one profile and returns whether it reached a healthy running state.
func (s *Service) RunForBisect(ctx context.Context, gameID, profileID string, direct bool) (bool, launch.Summary, error) {
	runCtx, cancel := context.WithTimeout(ctx, bisectRunTimeout)
	defer cancel()
	if err := s.start(runCtx, gameID, profileID, direct); err != nil {
		return false, launch.Summary{}, err
	}

	started := false
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
			if err := s.Stop(gameID); err != nil {
				return false, launch.Summary{}, err
			}
			return true, launch.Summary{}, nil
		case NoSteam:
			return false, launch.Summary{}, errors.New("the crash check needs a direct launch")
		case Failed:
			if !started {
				return false, launch.Summary{}, fmt.Errorf("bisect launch failed: %s", status.Error)
			}
			_, summary, err := s.LastRunSummary(gameID, profileID)
			return summaryHealthy(summary), summary, err
		case Idle:
			if started {
				_, summary, err := s.LastRunSummary(gameID, profileID)
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

func summaryHealthy(summary launch.Summary) bool {
	return !summary.Crashed && summary.Errors == 0
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
