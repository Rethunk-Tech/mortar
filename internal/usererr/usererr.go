package usererr

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"os"
	"syscall"
)

// Kind is a stable, frontend-parseable class for a user-facing error.
type Kind string

const (
	NotFound   Kind = "not_found"
	Busy       Kind = "busy"
	Network    Kind = "network"
	Permission Kind = "permission"
	DiskFull   Kind = "disk_full"
	Damaged    Kind = "damaged"
	Invalid    Kind = "invalid"
	Unknown    Kind = "unknown"
)

const prefix = "["
const mid = "] "

// Error is a Kind plus the original cause. errors.Is / errors.As see the cause.
type Error struct {
	kind Kind
	err  error
}

func (e *Error) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return prefix + string(e.kind) + mid + e.err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *Error) Kind() Kind {
	if e == nil {
		return Unknown
	}
	return e.kind
}

// Wrap tags err with kind. nil stays nil. An existing *Error keeps its cause and takes the new kind.
func Wrap(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{kind: kind, err: err}
}

// New is Wrap(kind, errors.New(msg)).
func New(kind Kind, msg string) error {
	return Wrap(kind, errors.New(msg))
}

// KindOf returns the tagged kind, else a stdlib classification, else Unknown.
func KindOf(err error) Kind {
	if err == nil {
		return Unknown
	}
	var ue *Error
	if errors.As(err, &ue) {
		return ue.kind
	}
	return classify(err)
}

func classify(err error) Kind {
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
		return NotFound
	}
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) {
		return Permission
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return Network
	}
	if isENOSPC(err) {
		return DiskFull
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return Network
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return Network
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return Network
	}
	return Unknown
}

func isENOSPC(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.ENOSPC {
		return true
	}
	return false
}

// Parse splits Error()'s "[kind] rest" form used on the Wails wire. Unknown if absent.
func Parse(s string) (kind Kind, raw string) {
	if len(s) < 4 || s[0] != '[' {
		return Unknown, s
	}
	for i := 1; i < len(s); i++ {
		if s[i] == ']' && i+1 < len(s) && s[i+1] == ' ' {
			k := Kind(s[1:i])
			switch k {
			case NotFound, Busy, Network, Permission, DiskFull, Damaged, Invalid:
				return k, s[i+2:]
			}
			return Unknown, s
		}
	}
	return Unknown, s
}
