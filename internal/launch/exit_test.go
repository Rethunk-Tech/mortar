package launch

import "testing"

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
