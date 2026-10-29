//go:build !windows

package launch

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// runs reports whether args start the program name, directly or through dotnet or mono.
func runs(args []string, name string) bool {
	is := func(a string) bool {
		// Wine and Proton processes carry the Windows path of the game as their program, and Windows names ignore
		// case. A Linux name is matched exactly: ignoring case there makes /bin/sh SUPERHOT's "SH".
		base := filepath.Base(strings.ReplaceAll(a, `\`, "/"))
		if ext := strings.ToLower(filepath.Ext(base)); ext == ".exe" || ext == ".dll" {
			return strings.EqualFold(strings.TrimSuffix(base, filepath.Ext(base)), name)
		}
		return base == name
	}
	if len(args) == 0 {
		return false
	}
	if is(args[0]) {
		return !systemProgram(args[0])
	}
	return hosted(args) && len(args) > 1 && is(args[1])
}

// hosted reports whether args run a program through dotnet or mono, which then is the second argument.
func hosted(args []string) bool {
	if len(args) < 2 {
		return false
	}
	host := filepath.Base(args[0])
	return host == "dotnet" || host == "mono"
}

// runsAny reports whether args start any of names.
func runsAny(args, names []string) bool {
	for _, name := range names {
		if runs(args, name) {
			return true
		}
	}
	return false
}

// Named reports whether p is one of names, as Processes matched it.
func (p Process) Named(names ...string) bool { return runsAny(p.Args, names) }

// Processes lists running processes whose executable is one of names, from procDir (/proc) on Linux, in one pass
// over the process table.
func Processes(procDir string, names ...string) ([]Process, error) {
	if procDir == "/proc" && sandbox.InFlatpak() {
		return hostProcesses(names)
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
		if !runsAny(args, names) {
			continue
		}
		p := Process{PID: pid, Args: args}
		// Unreadable for another user's process, and the launcher's Args still name the install then.
		p.Exe, _ = os.Readlink(filepath.Join(procDir, e.Name(), "exe"))
		// A program named without its folder ("sh") is a system one when its executable is. One named by a path is
		// judged by that path alone: Wine names the game's .exe, though its executable may be a system Wine's.
		if !strings.ContainsAny(args[0], `/\`) && systemProgram(p.Exe) && !hosted(args) {
			continue
		}
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

// hostScript lists every host process, since /proc inside the sandbox shows only Mortar's own. It runs two programs
// in all, not one per process: every profile change asks, and a desktop runs hundreds of processes. It prints
// "pid start-unix" lines (the proc entry's mtime is the start of a process that has not changed owner), a NUL, then
// grep's "pid/cmdline:arg" record per argument, NUL-terminated so an argument may hold any other byte.
const hostScript = `cd /proc && stat -c '%n %Y' [0-9]* 2>/dev/null; printf '\0'; grep -Hasz '' [0-9]*/cmdline 2>/dev/null; exit 0`

func hostProcesses(names []string) ([]Process, error) {
	out, err := hostOutput("sh", "-c", hostScript)
	if err != nil {
		return nil, err
	}
	return parseHostProcesses(string(out), names...), nil
}

func parseHostProcesses(out string, names ...string) []Process {
	starts, cmdlines, _ := strings.Cut(out, "\x00")
	start := map[int]time.Time{}
	for line := range strings.SplitSeq(starts, "\n") {
		pid, sec, _ := strings.Cut(line, " ")
		p, perr := strconv.Atoi(pid)
		s, serr := strconv.ParseInt(sec, 10, 64)
		if perr == nil && serr == nil {
			start[p] = time.Unix(s, 0)
		}
	}
	var procs []Process
	cur := Process{PID: -1}
	flush := func() {
		if cur.PID >= 0 && runsAny(cur.Args, names) {
			cur.Start = start[cur.PID]
			procs = append(procs, cur)
		}
	}
	for rec := range strings.SplitSeq(cmdlines, "\x00") {
		file, arg, ok := strings.Cut(rec, ":")
		pidText, isCmdline := strings.CutSuffix(file, "/cmdline")
		pid, err := strconv.Atoi(pidText)
		if !ok || !isCmdline || err != nil {
			continue
		}
		if pid != cur.PID {
			flush()
			cur = Process{PID: pid}
		}
		cur.Args = append(cur.Args, arg)
	}
	flush()
	return procs
}
