//go:build unix && !linux

package launch

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// WaitPID waits until pid exits and reports how, including processes Mortar did not spawn.
func WaitPID(pid int) (Exit, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return Exit{}, err
	}
	defer func() { _ = unix.Close(kq) }()
	change := unix.Kevent_t{
		Ident:  uint64(pid),
		Filter: unix.EVFILT_PROC,
		Flags:  unix.EV_ADD | unix.EV_ONESHOT,
		Fflags: unix.NOTE_EXIT | unix.NOTE_EXITSTATUS,
	}
	if _, err := unix.Kevent(kq, []unix.Kevent_t{change}, nil, nil); err != nil {
		return Exit{}, err
	}
	events := make([]unix.Kevent_t, 1)
	if _, err := unix.Kevent(kq, nil, events, nil); err != nil {
		return Exit{}, err
	}
	return waitStatusExit(syscall.WaitStatus(events[0].Data)), nil
}
