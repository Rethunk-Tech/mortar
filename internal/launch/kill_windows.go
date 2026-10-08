//go:build windows

package launch

import (
	"os"
	"sync"

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
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, id)
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	defer func() { _ = windows.CloseHandle(h) }()
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
