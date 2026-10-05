package launchsvc

import (
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// clockSlack tolerates a file clock a little behind the one that stamped the run.
const clockSlack = 2 * time.Second

// unityCrashed reports whether Unity's player log says the game crashed: its crash handler prints "Crash!!!" and a
// native failure prints "Fatal error". A log older than the run is a previous run's.
func unityCrashed(text string) bool {
	return strings.Contains(text, "Crash!!!") || strings.Contains(text, "Fatal error")
}

// playerCrashed reads the player log of the profile's loader, for loaders that have one and no crash line of their own.
func (s *Service) playerCrashed(gameID, profileID string, started time.Time) bool {
	l, ok := s.loaderOf(gameID, profileID)
	w, hasLog := l.(loader.WithPlayerLog)
	if !ok || !hasLog || s.settings == nil {
		return false
	}
	path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", w.PlayerLogRole())
	if err != nil {
		return false
	}
	st, err := fsx.Stat(path)
	if err != nil || st.ModTime().Before(started.Add(-clockSlack)) {
		return false
	}
	b, err := fsx.ReadFile(path)
	return err == nil && unityCrashed(string(b))
}
