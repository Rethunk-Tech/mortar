package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
)

func TestQuitWaitsUntilTheAppStopsAnswering(t *testing.T) {
	var methods []string
	answered := 0
	call := func(method string, _ control.Params, _ any, _ time.Duration) error {
		methods = append(methods, method)
		if method == "games" {
			answered++
			if answered > 2 {
				return controlwire.ErrNotRunning
			}
		}
		return nil
	}
	var o, e bytes.Buffer
	if code := run("9.9.9", call, []string{"quit"}, &o, &e); code != 0 {
		t.Fatalf("code %d stderr %q", code, e.String())
	}
	if methods[0] != "app.quit" || answered != 3 {
		t.Fatalf("calls %v", methods)
	}
}

func TestQuitWithNothingRunningSucceeds(t *testing.T) {
	call := func(string, control.Params, any, time.Duration) error { return controlwire.ErrNotRunning }
	var o, e bytes.Buffer
	if code := run("9.9.9", call, []string{"quit"}, &o, &e); code != 0 {
		t.Fatalf("code %d stderr %q", code, e.String())
	}
}
