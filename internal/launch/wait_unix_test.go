//go:build unix

package launch

import (
	"syscall"
	"testing"
)

func TestWaitStatusExit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ws   syscall.WaitStatus
		want Exit
	}{
		{"exit 3", 3 << 8, Exit{Code: 3}},
		{"clean exit", 0, Exit{}},
		{"SIGKILL", syscall.WaitStatus(syscall.SIGKILL), Exit{Code: 137, Signal: "SIGKILL"}},
		{"SIGSEGV", syscall.WaitStatus(syscall.SIGSEGV), Exit{Code: 139, Signal: "SIGSEGV"}},
	} {
		if got := waitStatusExit(tc.ws); got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
