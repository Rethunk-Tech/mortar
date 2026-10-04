package savessvc

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/usererr"
)

func TestErrBusyIsTaggedBusy(t *testing.T) {
	if usererr.KindOf(ErrBusy) != usererr.Busy {
		t.Fatalf("kind = %v", usererr.KindOf(ErrBusy))
	}
}
