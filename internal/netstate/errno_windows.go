package netstate

import "golang.org/x/sys/windows"

// Winsock reports a refused or timed-out connect with its own codes, which syscall.ECONNREFUSED and Errno.Timeout do
// not match on Windows.
const (
	refusedErrno  = windows.WSAECONNREFUSED
	timedOutErrno = windows.WSAETIMEDOUT
)
