package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func TestUpdateAllPrintsItsUndo(t *testing.T) {
	r := invoke(t, map[string]any{"updates.queue": control.QueuedUpdates{Queued: 2, Before: "ev1"}}, "update", "stardew", "My Farm", "--all")
	if r.code != 0 || !r.calls[0].params.All {
		t.Fatalf("call: %+v", r)
	}
	if !strings.Contains(r.out, "Queued 2 updates.") || !strings.Contains(r.out, `profile revert stardew "My Farm" ev1`) {
		t.Fatalf("out: %q", r.out)
	}
}
