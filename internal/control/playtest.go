package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

func (s *Services) playTest(ctx context.Context, game, profileID string) (launchsvc.TestLaunchResult, error) {
	if s.Launches == nil {
		return launchsvc.TestLaunchResult{}, fmt.Errorf("launch is unavailable")
	}
	return s.Launches.TestLaunch(ctx, game, profileID)
}
