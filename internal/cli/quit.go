package cli

import (
	"errors"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/controlwire"
)

// quitWait bounds how long quit waits for the app to stop answering; the Windows installer runs quit before it
// replaces or removes the exe and asks the user to close Mortar when this runs out.
const quitWait = 10 * time.Second

// quit asks the running app to close and returns once it no longer answers. With no app running it succeeds.
func (c *cmd) quit() error {
	err := c.call("app.quit", control.Params{}, nil, 5*time.Second)
	if errors.Is(err, controlwire.ErrNotRunning) {
		return nil
	}
	if err != nil {
		return err
	}
	for deadline := time.Now().Add(quitWait); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if err := c.call("games", control.Params{}, nil, time.Second); errors.Is(err, controlwire.ErrNotRunning) {
			return nil
		}
	}
	return errors.New("still open after 10s; Mortar may be asking whether to stop downloads or the game")
}
