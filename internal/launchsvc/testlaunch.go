package launchsvc

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

// TestLaunchResult is the outcome of a profile test launch.
type TestLaunchResult struct {
	ReachedTitle bool   `json:"reachedTitle"`
	Cause        string `json:"cause,omitempty"`
}

// TestLaunch saves nothing; it runs the profile the way Play would and reports title-screen or crash.
func (s *Service) TestLaunch(ctx context.Context, gameID, profileID string) (TestLaunchResult, error) {
	healthy, summary, err := s.RunForBisect(ctx, gameID, profileID)
	if err != nil {
		return TestLaunchResult{}, err
	}
	if healthy {
		return TestLaunchResult{ReachedTitle: true}, nil
	}
	return TestLaunchResult{Cause: testLaunchCause(summary)}, nil
}

func testLaunchCause(summary launch.Summary) string {
	if len(summary.Mods) > 0 {
		if summary.Mods[0].First != "" {
			return summary.Mods[0].First
		}
		if summary.Mods[0].Mod != "" {
			return summary.Mods[0].Mod
		}
	}
	if launch.ExitCrashed(summary.Exit) {
		return launch.DescribeExit(summary.Exit)
	}
	if summary.Crashed {
		return "crash"
	}
	return "did not reach the title screen"
}
