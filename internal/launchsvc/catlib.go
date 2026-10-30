package launchsvc

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-winsym/crashdump"
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

// fault is a crash record with what only a CatLib report carries: the caller of a fault raised in Windows or the C
// runtime, and the report's folder, which the window opens.
type fault struct {
	crashdump.Fault
	CalledFrom, ReportDir string
}

// catLib reads the newest crash report CatLib (a BepInEx 6 library for Cat Mail Co.) kept since `since` in a
// profile's BepInEx folder: BepInEx/CatLib/Crashes/<yyyy-MM-dd_HH-mm-ss>_<pid>/ holding report.txt and crash.dmp.
// The minidump, when present, names the module; the report's own lines fill in what it cannot.
func catLib(profileDir string, since time.Time) (fault, bool) {
	matches, _ := filepath.Glob(filepath.Join(profileDir, "BepInEx", "CatLib", "Crashes", "*_*"))
	var newest string
	var newestTime time.Time
	for _, dir := range matches {
		if info, err := os.Stat(dir); err == nil && info.IsDir() && info.ModTime().After(since) && info.ModTime().After(newestTime) {
			newest, newestTime = dir, info.ModTime()
		}
	}
	if newest == "" {
		return fault{}, false
	}
	d, _ := crashdump.ReadDump(filepath.Join(newest, "crash.dmp"))
	f := fault{Fault: d, ReportDir: newest}
	if text, err := readHead(newest, "report.txt"); err == nil {
		if r, ok := parseCatLibReport(text); ok {
			if f.Module == "" || crashdump.IsSystemModule(f.Module) {
				f.Module = r.Module
			}
			if f.Detail == "" {
				f.Detail = r.Detail
			}
			f.CalledFrom = r.CalledFrom
		}
	}
	return f, true
}

// readHead reads the first maxReportBytes of the file and drops the rest: a crash report is text, so the start of a
// damaged, huge one is still worth showing.
func readHead(dir, name string) (string, error) {
	f, err := os.OpenInRoot(dir, name)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxReportBytes))
	return strings.ToValidUTF8(string(b), ""), err
}

// parseCatLibReport reads report.txt's Module, Method and Called from lines. The module is the one the crash was
// raised in, or, when that is Windows or the C runtime, the caller that asked for it.
func parseCatLibReport(text string) (fault, bool) {
	m := catlibModule.FindStringSubmatch(text)
	if m == nil {
		return fault{}, false
	}
	f := fault{Module: baseName(m[1]), Detail: "exception " + strings.ToLower(m[3]) + " at " + m[2]}
	if meth := catlibMethod.FindStringSubmatch(text); meth != nil {
		f.Detail += ", " + strings.TrimSpace(meth[1])
	}
	if c := catlibCaller.FindStringSubmatch(text); c != nil {
		f.CalledFrom = strings.TrimSpace(c[1] + " + " + c[2] + "  " + c[3])
		if crashdump.IsSystemModule(f.Module) {
			f.Module = baseName(c[1])
		}
	}
	return f, true
}

// baseName is the file name of a Windows or Unix path, whichever separator it uses.
func baseName(p string) string {
	return p[strings.LastIndexAny(p, `\/`)+1:]
}
