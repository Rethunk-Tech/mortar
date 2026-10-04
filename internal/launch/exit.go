package launch

import "fmt"

// Exit is how the game process ended.
type Exit struct {
	Code    int    `json:"code"`
	Signal  string `json:"signal,omitempty"`
	Stopped bool   `json:"stopped,omitempty"`
}

// DescribeExit is the short clause for notices and CLI ("exited with code 134 (SIGABRT)").
func DescribeExit(x Exit) string {
	if x.Stopped {
		return "stopped"
	}
	if x.Signal != "" {
		if x.Code != 0 {
			return fmt.Sprintf("exited with code %d (%s)", x.Code, x.Signal)
		}
		return fmt.Sprintf("exited with %s", x.Signal)
	}
	if x.Code != 0 {
		return fmt.Sprintf("exited with code %d", x.Code)
	}
	return "exited with code 0"
}

// ExitCrashed is a non-zero code or a terminating signal that was not a user Stop.
func ExitCrashed(x Exit) bool {
	if x.Stopped {
		return false
	}
	return x.Code != 0 || x.Signal != ""
}

// ApplyExit sets Summary.Exit and marks Crashed when the process ended abnormally.
func ApplyExit(s *Summary, x Exit) {
	s.Exit = x
	if ExitCrashed(x) {
		s.Crashed = true
	}
}

// OutcomeOf is the stored run outcome from a failed start or a finished summary.
func OutcomeOf(failed, crashed bool) Outcome {
	if failed {
		return OutcomeFailed
	}
	if crashed {
		return OutcomeCrashed
	}
	return OutcomeRan
}
