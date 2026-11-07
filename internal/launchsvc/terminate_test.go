//go:build !windows

package launchsvc

import (
	"os/exec"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-gameproc"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

func TestTerminateLeavesAPidThatIsNoLongerTheGame(t *testing.T) {
	svc, _ := startEnv(t)
	sl := svc.selectedSlot(game.Find("stardew"))
	other := exec.CommandContext(t.Context(), "sleep", "30")
	if err := other.Start(); err != nil {
		t.Skip("no sleep program:", err)
	}
	// Wait reaps the child, so an ended process is seen as ended: a signal-0 probe still succeeds on a zombie.
	exited := make(chan struct{})
	go func() {
		_ = other.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		_ = other.Process.Kill()
		<-exited
	})
	// A listed game process whose pid has since been reused by an unrelated program.
	if err := svc.terminate(sl, gameproc.Process{PID: other.Process.Pid, Exe: "/usr/bin/sleep"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
		t.Fatal("the unrelated process was ended")
	case <-time.After(500 * time.Millisecond):
	}
}
