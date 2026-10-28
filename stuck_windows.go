//go:build windows

package main

import (
	"errors"
	"math"
	"os"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

func processExe(pid int) (string, error) {
	if pid <= 0 || pid > math.MaxUint32 {
		return "", errors.New("not a process id")
	}
	return launch.ImagePath(uint32(pid))
}

// waitGone waits up to 5 seconds for an ended process to exit.
func waitGone(p *os.Process) {
	done := make(chan struct{})
	go func() {
		_, _ = p.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func askEndStuck(msg string) bool {
	return w32.MessageBox(0, msg, "Mortar", mbYesNo|mbIconError|mbTopmost) == idYes
}
