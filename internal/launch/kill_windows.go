//go:build windows

package launch

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// jobs maps a started process's id to the job object holding it and its descendants.
var (
	jobsMu sync.Mutex
	jobs   = map[int]windows.Handle{}
)

// trackTree puts the started process into a job of its own, so killTree reaches what it spawns. Closing the job
// handle later does not end the process: the game outlives Mortar.
func trackTree(pid int) {
	id, err := pidUint32(pid)
	if err != nil {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, id)
	if err != nil {
		return
	}
	defer func() { _ = windows.CloseHandle(h) }()
	trackHandle(pid, h)
}

// trackHandle puts the process behind h into a job of its own and records the job under pid. The handle needs
// PROCESS_SET_QUOTA and PROCESS_TERMINATE.
func trackHandle(pid int, h windows.Handle) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	if windows.AssignProcessToJobObject(job, h) != nil {
		_ = windows.CloseHandle(job)
		return
	}
	jobsMu.Lock()
	jobs[pid] = job
	jobsMu.Unlock()
}

// untrackTree drops the job once its process has exited.
func untrackTree(pid int) {
	jobsMu.Lock()
	job, ok := jobs[pid]
	delete(jobs, pid)
	jobsMu.Unlock()
	if ok {
		_ = windows.CloseHandle(job)
	}
}

// terminateTracked ends the job holding pid and everything it started, and reports whether a job did.
func terminateTracked(pid int) bool {
	jobsMu.Lock()
	job, ok := jobs[pid]
	jobsMu.Unlock()
	return ok && windows.TerminateJobObject(job, 1) == nil
}

// killTree ends the process and, when it was tracked, everything it started.
func killTree(pid int) {
	jobsMu.Lock()
	job, ok := jobs[pid]
	jobsMu.Unlock()
	if ok && windows.TerminateJobObject(job, 1) == nil {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

// holdStart makes the process start suspended, so it is in its job before it can spawn anything that would escape.
func holdStart(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
}

// releaseStart resumes the one thread of a process holdStart started.
func releaseStart(pid int) error {
	id, err := pidUint32(pid)
	if err != nil {
		return err
	}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snap, &entry); err == nil; err = windows.Thread32Next(snap, &entry) {
		if entry.OwnerProcessID != id {
			continue
		}
		th, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		defer func() { _ = windows.CloseHandle(th) }()
		_, err = windows.ResumeThread(th)
		return err
	}
	return fmt.Errorf("no thread to resume in process %d", pid)
}
