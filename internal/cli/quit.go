package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// quitWait bounds how long quit waits for the app to exit; the Windows installer runs quit before it replaces or
// removes the exe and asks the user to close Mortar when this runs out.
const quitWait = 10 * time.Second

// processAlive is how quit tells the app's process has ended. Each Flatpak run has its own PID namespace, so a pid
// from the app means nothing to a CLI started by another flatpak run; there only the control channel can tell.
var processAlive = func(pid int) bool { return !sandbox.InFlatpak() && launch.Alive(pid) }

// quit asks the running app to close and returns once its process has exited, since the control channel closes
// before the app finishes shutting down. With no app running it succeeds.
func (c *cmd) quit() error {
	var reply any
	err := c.call("app.quit", control.Params{Force: c.force}, &reply, 5*time.Second)
	if errors.Is(err, controlwire.ErrNotRunning) {
		return nil
	}
	if err != nil {
		return err
	}
	pid := 0
	if n, ok := reply.(float64); ok {
		pid = int(n)
	}
	for deadline := time.Now().Add(quitWait); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if err := c.call("games", control.Params{}, nil, time.Second); !errors.Is(err, controlwire.ErrNotRunning) {
			continue
		}
		if pid <= 0 || !processAlive(pid) {
			return nil
		}
	}
	if pid > 0 && processAlive(pid) {
		return fmt.Errorf("the app (pid %d) is still running %s after it was asked to quit", pid, quitWait)
	}
	return fmt.Errorf("the app still answers %s after it was asked to quit", quitWait)
}
