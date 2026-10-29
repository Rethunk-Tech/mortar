// Package wineoverride is what Mortar does for a loader that runs under Wine: it edits a Proton prefix's registry so
// Wine loads the loader's proxy library natively, and writes host paths the way Wine reads them.
package wineoverride

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const (
	dllOverridesHeader = `[Software\\Wine\\DllOverrides]`
	backupSuffix       = ".mortar-backup"
	// filetimeEpoch is the seconds between 1601 and 1970, for Wine's #time= lines.
	filetimeEpoch = 11644473600
)

// Ensure makes the Proton prefix's user.reg load the library dll (such as winhttp, which Doorstop injects through, or
// version, which MelonLoader does) natively. It is idempotent,
// keeps the file's line endings and other sections, and copies the file to user.reg.mortar-backup once before the
// first change. Run it while Wine is not running, or wineserver writes its own copy back over it.
func Ensure(userReg, dll string) error {
	raw, err := fsx.ReadFile(userReg)
	if err != nil {
		return err
	}
	text := string(raw)
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(text, eol)
	updated, changed := withOverride(lines, dll, nowFiletime)
	if !changed {
		return nil
	}
	backup := userReg + backupSuffix
	if _, err := os.Stat(backup); errors.Is(err, fs.ErrNotExist) {
		if err := fsx.WriteFile(backup, raw, 0o600); err != nil {
			return err
		}
	}
	return datadir.WriteFileAtomic(userReg, []byte(strings.Join(updated, eol)), 0o600)
}

// withOverride returns lines with the override for dll in the DllOverrides section, adding the section when it is missing.
func withOverride(lines []string, dll string, now func() (unix int64)) ([]string, bool) {
	line := fmt.Sprintf(`"%s"="native,builtin"`, dll)
	start := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.ToLower(l), strings.ToLower(dllOverridesHeader)) })
	if start < 0 {
		unix := now()
		filetime := (unix + filetimeEpoch) * 10_000_000
		out := slices.Clone(lines)
		// A file ends with a newline, so its last element is empty and the blank line before a section already exists.
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
		return append(out, "", fmt.Sprintf("%s %d", dllOverridesHeader, unix), fmt.Sprintf("#time=%x", filetime), line, ""), true
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(lines[end], "[") && lines[end] != "" {
		end++
	}
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(strings.ToLower(lines[i]), `"`+strings.ToLower(dll)+`"=`) {
			if lines[i] == line {
				return lines, false
			}
			out := slices.Clone(lines)
			out[i] = line
			return out, true
		}
	}
	out := slices.Clone(lines[:end])
	out = append(out, line)
	return append(out, lines[end:]...), true
}

func nowFiletime() int64 { return time.Now().Unix() }

// ZPath is the absolute host path p as the game under Wine reads it, on the Z: drive that maps the host's root.
func ZPath(p string) string {
	return `Z:` + strings.ReplaceAll(filepath.ToSlash(p), "/", `\`)
}
