//go:build linux

package launch

import (
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// WaitPID waits until pid exits and reports how, including processes Mortar did not spawn.
func WaitPID(pid int) (Exit, error) {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		for Alive(pid) {
			time.Sleep(100 * time.Millisecond)
		}
		return Exit{}, err
	}
	defer func() { _ = unix.Close(fd) }()
	var info unix.Siginfo
	if err := unix.Waitid(unix.P_PIDFD, fd, &info, unix.WEXITED, nil); err != nil {
		return Exit{}, err
	}
	return waitidExit(&info), nil
}

func waitidExit(info *unix.Siginfo) Exit {
	type layout struct {
		Signo  int32
		Errno  int32
		Code   int32
		Pad    int32
		Pid    int32
		Uid    uint32
		Status int32
	}
	p := (*layout)(unsafe.Pointer(info))
	const (
		cldKilled = 2
		cldDumped = 3
	)
	switch p.Code {
	case cldKilled, cldDumped:
		sig := syscall.Signal(p.Status)
		return Exit{Code: 128 + int(p.Status), Signal: unixSignalName(sig)}
	default:
		return Exit{Code: int(p.Status)}
	}
}
