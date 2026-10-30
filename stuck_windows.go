//go:build windows

package main

import (
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/w32"
)

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
