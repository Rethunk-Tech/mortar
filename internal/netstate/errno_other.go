//go:build !windows

package netstate

import "syscall"

const (
	refusedErrno  = syscall.ECONNREFUSED
	timedOutErrno = syscall.ETIMEDOUT
)
