//go:build !windows

package folder_test

import (
	"syscall"
	"time"
)

func cpu() time.Duration {
	var r syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &r)
	return time.Duration(r.Utime.Nano() + r.Stime.Nano())
}
