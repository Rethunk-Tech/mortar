//go:build !windows

package netstate

import "syscall"

const (
	connectCall = "connect"
	osRefused   = syscall.ECONNREFUSED
	osTimedOut  = syscall.ETIMEDOUT
)
