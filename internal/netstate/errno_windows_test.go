package netstate

import "syscall"

// What a failed connect on Windows wraps: Winsock's WSAECONNREFUSED and WSAETIMEDOUT, from connectex.
const (
	connectCall = "connectex"
	osRefused   = syscall.Errno(10061)
	osTimedOut  = syscall.Errno(10060)
)
