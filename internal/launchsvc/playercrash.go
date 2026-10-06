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
// native failure prints "Fatal error".
func unityCrashed(text string) bool {
	return strings.Contains(text, "Crash!!!") || strings.Contains(text, "Fatal error")
}

// playerLogSince is the Unity player log of the profile's loader, for loaders that read one, when it was written after
// since. The game keeps one player log and starts it afresh each launch, so a log older than the run is a previous
// run's.
func (s *Service) playerLogSince(gameID, profileID string, since time.Time) (string, bool) {
	l, ok := s.loaderOf(gameID, profileID)
	w, hasLog := l.(loader.WithPlayerLog)
	if !ok || !hasLog || s.settings == nil {
		return "", false
	}
	path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", w.PlayerLogRole())
	if err != nil {
		return "", false
	}
	st, err := fsx.Stat(path)
	if err != nil || st.ModTime().Before(since.Add(-clockSlack)) {
		return "", false
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.ToValidUTF8(string(b), ""), true
}
