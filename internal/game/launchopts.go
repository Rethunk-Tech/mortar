package game

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/steam"
)

// LaunchOptions returns the launch options Steam holds for the game, or "" when none are set or Steam is missing.
func (s *Service) LaunchOptions(id string) (string, error) {
	g := Find(id)
	if g == nil {
		return "", fmt.Errorf("unknown game %q", id)
	}
	st, status := steam.Locate(s.home)
	if status != steam.Found {
		return "", nil
	}
	return st.LaunchOptions(g.SteamAppID())
}
