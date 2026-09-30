//go:build !windows

package launch

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Processes lists running processes whose executable is name, from procDir (/proc) on Linux.
func Processes(procDir, name string) ([]Process, error) {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, err
	}
	var out []Process
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		raw, err := fsx.ReadFile(filepath.Join(procDir, e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(string(bytes.TrimRight(raw, "\x00")), "\x00")
		if !runs(args, name) {
			continue
		}
		p := Process{PID: pid, Args: args}
		p.Start = startTime(procDir, e.Name())
		out = append(out, p)
	}
	return out, nil
}

// clockTicks is the kernel's USER_HZ, 100 on every Linux platform Go supports.
const clockTicks = 100

// startTime is when the process began, from its /proc/<pid>/stat start tick and the boot time in /proc/stat;
// the zero time when either is unreadable.
func startTime(procDir, pid string) time.Time {
	stat, err := fsx.ReadFile(filepath.Join(procDir, pid, "stat"))
	if err != nil {
		return time.Time{}
	}
	// The command name may hold spaces and parentheses, so fields count from the last ')'.
	fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+1:])
	if len(fields) < 20 {
		return time.Time{}
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}
	}
	sys, err := fsx.ReadFile(filepath.Join(procDir, "stat"))
	if err != nil {
		return time.Time{}
	}
	for line := range strings.SplitSeq(string(sys), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			boot, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return time.Time{}
			}
			return time.Unix(boot+ticks/clockTicks, 0)
		}
	}
	return time.Time{}
}
