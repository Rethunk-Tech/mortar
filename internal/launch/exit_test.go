package launch

import "testing"

func TestClassifyWait(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		wait     func() Exit
		stopped  bool
		crashed  bool
		code     int
		signal   string
		stoppedX bool
	}{
		{name: "code 0", wait: func() Exit { return Exit{Code: 0} }, crashed: false, code: 0},
		{name: "code 1", wait: func() Exit { return Exit{Code: 1} }, crashed: true, code: 1},
		{name: "code 134", wait: func() Exit { return Exit{Code: 134, Signal: "SIGABRT"} }, crashed: true, code: 134, signal: "SIGABRT"},
		{name: "SIGKILL", wait: func() Exit { return Exit{Code: 137, Signal: "SIGKILL"} }, crashed: true, code: 137, signal: "SIGKILL"},
		{name: "SIGABRT", wait: func() Exit { return Exit{Signal: "SIGABRT"} }, crashed: true, signal: "SIGABRT"},
		{name: "user Stop", wait: func() Exit { return Exit{Code: 137, Signal: "SIGKILL"} }, stopped: true, crashed: false, code: 137, signal: "SIGKILL", stoppedX: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, crashed := ClassifyWait(tc.wait, tc.stopped)
			if crashed != tc.crashed || got.Code != tc.code || got.Signal != tc.signal || got.Stopped != tc.stoppedX {
				t.Fatalf("got %#v crashed=%v", got, crashed)
			}
		})
	}
}

func TestApplyExitAndOutcome(t *testing.T) {
	t.Parallel()
	s := Summarize("")
	if s.Crashed || OutcomeOf(false, s.Crashed) != OutcomeRan {
		t.Fatalf("clean log: %#v", s)
	}
	ApplyExit(&s, Exit{Code: 1})
	if !s.Crashed || s.Exit.Code != 1 || OutcomeOf(false, s.Crashed) != OutcomeCrashed {
		t.Fatalf("code 1: %#v", s)
	}
	stopped := Summarize("")
	ApplyExit(&stopped, Exit{Code: 9, Signal: "SIGKILL", Stopped: true})
	if stopped.Crashed || OutcomeOf(false, stopped.Crashed) != OutcomeRan {
		t.Fatalf("stop: %#v", stopped)
	}
	if OutcomeOf(true, true) != OutcomeFailed {
		t.Fatal("failed start wins")
	}
}

func TestDescribeExit(t *testing.T) {
	t.Parallel()
	if got := DescribeExit(Exit{Code: 134, Signal: "SIGABRT"}); got != "exited with code 134 (SIGABRT)" {
		t.Fatal(got)
	}
	if got := DescribeExit(Exit{Stopped: true, Code: 9, Signal: "SIGKILL"}); got != "stopped" {
		t.Fatal(got)
	}
}
