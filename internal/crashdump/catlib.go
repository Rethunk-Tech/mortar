package crashdump

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxReportBytes = 256 << 10

var (
	// CatLib writes these lines at the top of report.txt (CatLib src/CatLib/Diagnostics/CrashText.cs Report):
	// "Module: KERNELBASE.dll + 0x1c408a, exception 0xc0000005", "Method: RaiseException + 0x8a" and
	// "Called from: UnityPlayer.dll + 0xdb437  Utils_CUSTOM_ForceCrash + 0x17, the first frame outside ...".
	catlibModule = regexp.MustCompile(`(?m)^Module: (\S+) \+ (0x[0-9a-fA-F]+), exception (0x[0-9a-fA-F]+)`)
	catlibMethod = regexp.MustCompile(`(?m)^Method: ([^\r\n]+)`)
	catlibCaller = regexp.MustCompile(`(?m)^Called from: (\S+) \+ (0x[0-9a-fA-F]+)(?:\s+([^,\r\n]+))?`)
)

// CatLib reads the newest crash report CatLib (a BepInEx 6 library for Cat Mail Co.) kept since `since` in a
// profile's BepInEx folder: BepInEx/CatLib/Crashes/<yyyy-MM-dd_HH-mm-ss>_<pid>/ holding report.txt and crash.dmp.
// The minidump, when present, names the module; the report's own lines fill in what it cannot.
func CatLib(profileDir string, since time.Time) (Fault, bool) {
	matches, _ := filepath.Glob(filepath.Join(profileDir, "BepInEx", "CatLib", "Crashes", "*_*"))
	var newest string
	var newestTime time.Time
	for _, dir := range matches {
		if info, err := os.Stat(dir); err == nil && info.IsDir() && info.ModTime().After(since) && info.ModTime().After(newestTime) {
			newest, newestTime = dir, info.ModTime()
		}
	}
	if newest == "" {
		return Fault{}, false
	}
	fault, _ := readDump(filepath.Join(newest, "crash.dmp"))
	if text, err := readCapped(newest, "report.txt"); err == nil {
		if r, ok := ParseCatLibReport(text); ok {
			if fault.Module == "" || isSystemModule(fault.Module) {
				fault.Module = r.Module
			}
			if fault.Detail == "" {
				fault.Detail = r.Detail
			}
			fault.CalledFrom = r.CalledFrom
		}
	}
	fault.ReportDir = newest
	return fault, true
}

func readCapped(dir, name string) (string, error) {
	f, err := os.OpenInRoot(dir, name)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxReportBytes))
	return strings.ToValidUTF8(string(b), ""), err
}

// ParseCatLibReport reads report.txt's Module, Method and Called from lines. The module is the one the crash was
// raised in, or, when that is Windows or the C runtime, the caller that asked for it.
func ParseCatLibReport(text string) (Fault, bool) {
	m := catlibModule.FindStringSubmatch(text)
	if m == nil {
		return Fault{}, false
	}
	f := Fault{Module: baseName(m[1]), Detail: "exception " + strings.ToLower(m[3]) + " at " + m[2]}
	if meth := catlibMethod.FindStringSubmatch(text); meth != nil {
		f.Detail += ", " + strings.TrimSpace(meth[1])
	}
	if c := catlibCaller.FindStringSubmatch(text); c != nil {
		f.CalledFrom = strings.TrimSpace(c[1] + " + " + c[2] + "  " + c[3])
		if isSystemModule(f.Module) {
			f.Module = baseName(c[1])
		}
	}
	return f, true
}

func isSystemModule(name string) bool {
	return systemModules[strings.ToLower(strings.TrimSuffix(name, ".dll"))] || systemModules[name]
}
