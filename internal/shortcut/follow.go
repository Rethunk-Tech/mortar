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
// one has to last as long as the game for Steam to count the playtime. A game that is not running within patience
// (Play blocked, then cancelled) ends the wait with ErrNeverStarted; a Mortar that stops answering ends it too.
func Follow(dir string, r Request, every, patience time.Duration) error {
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
		case ran:
			return nil
		case time.Now().After(deadline):
			return ErrNeverStarted
		}
		time.Sleep(every)
	}
}
