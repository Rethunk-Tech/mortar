package crashdump

import (
	"regexp"
	"strings"
)

var (
	// unityFrame is a native frame of a Unity error.log stack: "0x00007FF8A1B2C3D4 (UnityPlayer) UnityMain".
	unityFrame = regexp.MustCompile(`^\s*0x[0-9A-Fa-f]+ \(([^)]+)\)`)
	// coreFrame is a frame of coredumpctl's report: "#1  0x00007f12 name (libmono-2.0.so + 0x1234)".
	coreFrame = regexp.MustCompile(`^\s*#\d+\s+0x[0-9a-f]+\s+.*\(([^()\s]+) \+ 0x[0-9a-f]+\)`)
	exception = regexp.MustCompile(`(?m)^\s*(EXCEPTION_[A-Z_]+|SIG[A-Z]+)\b`)
)

// systemModules are the libraries a crash passes through on its way down; none of them is where it started.
var systemModules = map[string]bool{
	"ntdll": true, "kernel32": true, "kernelbase": true, "ucrtbase": true, "vcruntime140": true, "msvcrt": true,
	"libc.so.6": true, "libpthread.so.0": true, "libm.so.6": true, "libgcc_s.so.1": true, "ld-linux-x86-64.so.2": true,
}

// ParseUnityErrorLog names the module of the first native frame in a Unity error.log, the top of the crashing stack.
func ParseUnityErrorLog(text string) (Fault, bool) {
	return firstFrame(text, unityFrame)
}

// ParseCoredumpInfo names the module of the first non-system frame of `coredumpctl info` output.
func ParseCoredumpInfo(text string) (Fault, bool) {
	return firstFrame(text, coreFrame)
}

func firstFrame(text string, frame *regexp.Regexp) (Fault, bool) {
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r", ""), "\n") {
		m := frame.FindStringSubmatch(line)
		if m == nil || systemModules[strings.ToLower(strings.TrimSuffix(m[1], ".dll"))] || systemModules[m[1]] {
			continue
		}
		fault := Fault{Module: baseName(m[1])}
		if e := exception.FindString(text); e != "" {
			fault.Detail = strings.TrimSpace(e)
		}
		return fault, true
	}
	return Fault{}, false
}
