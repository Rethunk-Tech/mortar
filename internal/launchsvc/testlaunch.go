package launchsvc

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// TestLaunchResult is the outcome of a profile test launch.
type TestLaunchResult struct {
	ReachedTitle bool   `json:"reachedTitle"`
	Cause        string `json:"cause,omitempty"`
}

// TestLaunch saves nothing; it runs the profile the way Play would, on the install with this id when given, and
// reports title-screen or crash.
func (s *Service) TestLaunch(ctx context.Context, gameID, profileID, installID string) (TestLaunchResult, error) {
	return s.testLaunch(ctx, gameID, profileID, installID)
}

// TestLaunchWith is TestLaunch with exactly these launch options, prefix and environment, whatever preset is the
// profile's default.
func (s *Service) TestLaunchWith(ctx context.Context, gameID, profileID, options, prefix, env string) (TestLaunchResult, error) {
	spec := profile.LaunchSpec{Name: profile.BasePresetName, Options: options, Prefix: prefix, Env: env}
	return s.testLaunch(context.WithValue(ctx, specKey{}, spec), gameID, profileID, "")
}

func (s *Service) testLaunch(ctx context.Context, gameID, profileID, installID string) (TestLaunchResult, error) {
	healthy, summary, err := s.runForInstall(ctx, gameID, profileID, installID)
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
