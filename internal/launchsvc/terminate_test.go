//go:build !windows

package launchsvc

import (
	"os/exec"
	"syscall"
	"testing"

	"github.com/Rethunk-Tech/go-gameproc"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

func TestTerminateLeavesAPidThatIsNoLongerTheGame(t *testing.T) {
	svc, _ := startEnv(t)
	sl := svc.selectedSlot(game.Find("stardew"))
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Skip("no sleep program:", err)
	}
	t.Cleanup(func() {
		_ = other.Process.Kill()
		_ = other.Wait()
	})
	// A listed game process whose pid has since been reused by an unrelated program.
	if err := svc.terminate(sl, gameproc.Process{PID: other.Process.Pid, Exe: "/usr/bin/sleep"}); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(other.Process.Pid, 0); err != nil {
		t.Fatalf("the unrelated process was ended: %v", err)
	}
}
