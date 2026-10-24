package shortcut

import (
	"errors"
	"fmt"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
)

// PlayRequestMethod is the control method that hands a play request to the running Mortar's window.
const PlayRequestMethod = "play.request"

// ErrNeverStarted means the game a Steam session asked for did not start in time, as when the user cancelled Play.
var ErrNeverStarted = errors.New("the game did not start")

// Follow hands a Steam session's request to the Mortar running with data folder dir, then waits until the run it
// starts ends, polling the game's launch status every so often. Steam tracks only the process it started, so this
// one has to last as long as the game for Steam to count the playtime. A run that starts and ends between two polls
// is seen by the run it recorded. A game that is not running within patience (Play blocked, then cancelled) ends the
// wait with ErrNeverStarted; a Mortar that stops answering ends it too.
func Follow(dir string, r Request, every, patience time.Duration) error {
	before := runCount(dir, r)
	if err := controlwire.CallDir(dir, PlayRequestMethod, r, nil, 5*time.Second); err != nil {
		return err
	}
	ran := false
	deadline := time.Now().Add(patience)
	for {
		var st launchsvc.Status
		if err := controlwire.CallDir(dir, "status", map[string]string{"game": r.Game}, &st, 5*time.Second); err != nil {
			return fmt.Errorf("following %s: %w", r.Game, err)
		}
		switch {
		case st.State.Active():
			ran = true
		case ran, before >= 0 && runCount(dir, r) > before:
			return nil
		case time.Now().After(deadline):
			return ErrNeverStarted
		}
		time.Sleep(every)
	}
}

// runCount is how many runs the profile has recorded, or -1 when the running Mortar cannot say.
func runCount(dir string, r Request) int {
	var runs []launchsvc.Run
	if err := controlwire.CallDir(dir, "runs", map[string]string{"game": r.Game, "profile": r.Profile}, &runs, 5*time.Second); err != nil {
		return -1
	}
	return len(runs)
}
