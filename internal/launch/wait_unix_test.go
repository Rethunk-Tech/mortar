//go:build unix

package launch

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-gameproc"
)

// A process that is not Mortar's child reports no exit status, and launchsvc relies on the error to tell that from a clean exit.
func TestWaitPIDOfANonChildReturnsAnError(t *testing.T) {
	t.Parallel()
	// The shell starts sleep in the background and exits; sleep is then reparented, not our child.
	out, err := exec.CommandContext(t.Context(), "sh", "-c", "sleep 0.3 & echo $!").Output()
	if err != nil {
		t.Fatal(err)
	}
	var pid int
	if _, err := fmt.Sscan(string(out), &pid); err != nil || pid <= 0 {
		t.Fatalf("pid %q: %v", out, err)
	}
	done := make(chan error, 1)
	go func() { _, err := WaitPID(pid); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, gameproc.ErrNotChild) {
			t.Fatalf("WaitPID of a non-child returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("WaitPID did not return")
	}
}
