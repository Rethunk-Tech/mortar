package crashdump

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxDumpBytes bounds the dumps read: a full-memory dump is gigabytes, and only its header streams are needed, so one
// larger than this is skipped rather than risked.
const maxDumpBytes = 4 << 30

// Windows reads the dumps a crash since `since` left in a Windows user's profile: WER LocalDumps (crashDumps, named for
// the exe) and Unity crash folders under temp (<company>/<product>/Crashes/Crash_*). exes are the game's executable
// names. A Unity folder's error.log is read first, since it names the module itself.
func Windows(crashDumps, temp string, exes []string, since time.Time) []Fault {
	var out []Fault
	for _, dir := range unityCrashDirs(temp, since) {
		if b, err := os.ReadFile(filepath.Join(dir, "error.log")); err == nil {
			if f, ok := ParseUnityErrorLog(string(b)); ok {
				out = append(out, f)
				continue
			}
		}
		if f, ok := readDump(filepath.Join(dir, "crash.dmp")); ok {
			out = append(out, f)
		}
	}
	for _, exe := range exes {
		stem := strings.TrimSuffix(exe, filepath.Ext(exe))
		matches, _ := filepath.Glob(filepath.Join(crashDumps, stem+"*.dmp"))
		for _, path := range matches {
			if info, err := os.Stat(path); err != nil || !info.ModTime().After(since) {
				continue
			}
			if f, ok := readDump(path); ok {
				out = append(out, f)
			}
		}
	}
	return out
}

func unityCrashDirs(temp string, since time.Time) []string {
	matches, _ := filepath.Glob(filepath.Join(temp, "*", "*", "Crashes", "Crash_*"))
	var out []string
	for _, dir := range matches {
		if info, err := os.Stat(dir); err == nil && info.IsDir() && info.ModTime().After(since) {
			out = append(out, dir)
		}
	}
	return out
}

func readDump(path string) (Fault, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Fault{}, false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || info.Size() > maxDumpBytes {
		return Fault{}, false
	}
	fault, err := ParseMinidump(f, info.Size())
	return fault, err == nil
}
