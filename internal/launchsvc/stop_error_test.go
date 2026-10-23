package launchsvc

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestARefusedStopIsNotAStoragePermissionError(t *testing.T) {
	t.Parallel()
	refused := fmt.Errorf("Access is denied.: %w", fs.ErrPermission)
	if usererr.KindOf(refused) != usererr.Permission {
		t.Fatal("the premise: a refused stop classifies as a storage permission error")
	}
	got := errors.Join(nil, stopError(refused))
	if k := usererr.KindOf(got); k != usererr.ProcessDenied {
		t.Fatalf("kind = %q", k)
	}
	other := errors.New("boom")
	if !errors.Is(stopError(other), other) || stopError(nil) != nil {
		t.Fatal("other errors must pass through")
	}
}
