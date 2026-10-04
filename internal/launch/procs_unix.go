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
	"github.com/Rethunk-AI/mortar/internal/sandbox"
)

// runs reports whether args start the program name, directly or through dotnet or mono.
func runs(args []string, name string) bool {
	is := func(a string) bool {
		base := filepath.Base(a)
		return strings.EqualFold(strings.TrimSuffix(strings.TrimSuffix(base, ".exe"), ".dll"), name)
	}
	if len(args) == 0 {
		return false
	}
	if is(args[0]) {
		return true
	}
	host := filepath.Base(args[0])
	return (host == "dotnet" || host == "mono") && len(args) > 1 && is(args[1])
}

// Processes lists running processes whose executable is name, from procDir (/proc) on Linux.
func Processes(procDir, name string) ([]Process, error) {
	if procDir == "/proc" && sandbox.InFlatpak() {
		return hostProcesses(name)
	}
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

// hostScript prints "pid<TAB>start-unix<TAB>arg<US>arg..." for every host process; /proc inside the sandbox
// shows only Mortar's own. The proc entry's mtime is the process start for a process that has not changed owner.
const hostScript = `for d in /proc/[0-9]*; do [ -r "$d/cmdline" ] || continue; printf '%s\t%s\t' "${d#/proc/}" "$(stat -c %Y "$d")"; tr '\0' '\037' <"$d/cmdline"; echo; done`

func hostProcesses(name string) ([]Process, error) {
	out, err := sandbox.HostOutput("sh", "-c", hostScript)
	if err != nil {
		return nil, err
	}
	return parseHostProcesses(string(out), name), nil
}

func parseHostProcesses(out, name string) []Process {
	var procs []Process
	for line := range strings.SplitSeq(out, "\n") {
		f := strings.SplitN(line, "\t", 3)
		if len(f) != 3 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(f[2], "\x1f"), "\x1f")
		if !runs(args, name) {
			continue
		}
		p := Process{PID: pid, Args: args}
		if sec, err := strconv.ParseInt(f[1], 10, 64); err == nil {
			p.Start = time.Unix(sec, 0)
		}
		procs = append(procs, p)
	}
	return procs
}
