package launch

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// exitCode is a process exit status for a start that did not go through os/exec.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit status %d", int(e)) }

// ExitCode is read by WaitError and waitCode.
func (e exitCode) ExitCode() int { return int(e) }

// startOwnConsole starts a program in a new console that it reads and writes itself, as Vortex and Steam start SMAPI.
// os/exec cannot: it always passes standard handles (STARTF_USESTDHANDLES), and when none are set they are NUL, so
// SMAPI's console stayed empty and its input loop read end-of-file.
func startOwnConsole(env []string, dir, name string, args []string) (<-chan error, error) {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, syscall.EscapeArg(name))
	for _, a := range args {
		parts = append(parts, syscall.EscapeArg(a))
	}
	cmdLine, err := windows.UTF16PtrFromString(strings.Join(parts, " "))
	if err != nil {
		return nil, err
	}
	app, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	var cwd *uint16
	if dir != "" {
		if cwd, err = windows.UTF16PtrFromString(dir); err != nil {
			return nil, err
		}
	}
	flags := uint32(windows.CREATE_NEW_CONSOLE | windows.CREATE_SUSPENDED)
	var block *uint16
	if len(env) > 0 {
		flags |= windows.CREATE_UNICODE_ENVIRONMENT
		block = envBlock(append(os.Environ(), env...))
	}
	si := windows.StartupInfo{}
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(app, cmdLine, nil, nil, false, flags, block, cwd, &si, &pi); err != nil {
		return nil, &os.PathError{Op: "start", Path: name, Err: err}
	}
	pid := int(pi.ProcessId)
	trackHandle(pid, pi.Process)
	_, resumeErr := windows.ResumeThread(pi.Thread)
	_ = windows.CloseHandle(pi.Thread)
	if resumeErr != nil {
		killTree(pid)
		untrackTree(pid)
		_ = windows.CloseHandle(pi.Process)
		return nil, &os.PathError{Op: "start", Path: name, Err: resumeErr}
	}
	done := make(chan error, 1)
	go func() {
		defer func() { _ = windows.CloseHandle(pi.Process) }()
		defer untrackTree(pid)
		if _, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE); err != nil {
			done <- err
			return
		}
		var code uint32
		if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
			done <- err
			return
		}
		if code == 0 {
			done <- nil
			return
		}
		done <- exitCode(code)
	}()
	return done, nil
}

// envBlock is env as a sorted, NUL-separated, double-NUL-terminated UTF-16 block, as CreateProcess wants it; a
// later duplicate of a variable wins, as with os/exec.
func envBlock(env []string) *uint16 {
	last := map[string]string{}
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		last[strings.ToUpper(k)] = kv
	}
	keys := make([]string, 0, len(last))
	for k := range last {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []uint16
	for _, k := range keys {
		out = append(out, utf16.Encode([]rune(last[k]))...)
		out = append(out, 0)
	}
	out = append(out, 0)
	return &out[0]
}
