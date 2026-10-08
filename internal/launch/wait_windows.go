//go:build windows

package launch

import (
	"errors"

	"golang.org/x/sys/windows"
)

// WaitError maps cmd.Wait's result to an Exit.
func WaitError(err error) Exit {
	if err == nil {
		return Exit{Code: 0}
	}
	if ee, ok := errors.AsType[interface {
		error
		ExitCode() int
	}](err); ok {
		return Exit{Code: ee.ExitCode()}
	}
	return Exit{Code: 1}
}

// WaitPID waits until pid exits and reports its code.
func WaitPID(pid int) (Exit, error) {
	id, err := pidUint32(pid)
	if err != nil {
		return Exit{}, err
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, id)
	if err != nil {
		return Exit{}, err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if _, err := windows.WaitForSingleObject(h, windows.INFINITE); err != nil {
		return Exit{}, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return Exit{}, err
	}
	return Exit{Code: int(code)}, nil
}
