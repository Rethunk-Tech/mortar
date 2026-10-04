package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// maxLogBytes caps mortar.log so a long session cannot fill the disk; the previous run's log is kept separately.
const maxLogBytes = 10 << 20

// maxCrashBytes caps crash.log at startup once crash.seen already covers the file.
const maxCrashBytes = 1 << 20

// cappedWriter passes writes through until limit bytes, then writes one notice and drops the rest.
type cappedWriter struct {
	mu    sync.Mutex
	w     io.Writer
	left  int64
	ended bool
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ended && int64(len(p)) > c.left {
		c.ended = true
		if _, err := io.WriteString(c.w, "mortar.log reached its size limit; later lines go to stderr only\n"); err != nil {
			return len(p), err
		}
	}
	if c.ended {
		// The clean-shutdown record is what the next start reads to tell a quit from a crash, so it outlives the cap.
		if bytes.Contains(p, []byte("msg=shutdown")) {
			return c.w.Write(p)
		}
		return len(p), nil
	}
	c.left -= int64(len(p))
	return c.w.Write(p)
}

func capCrashLog(dataDir string) bool {
	return capCrashLogAt(dataDir, maxCrashBytes)
}

// capCrashLogAt empties crash.log when it is over limit and already seen, and reports whether it did.
func capCrashLogAt(dataDir string, limit int64) bool {
	path := filepath.Join(dataDir, "crash.log")
	info, err := os.Stat(path)
	if err != nil || info.Size() <= limit {
		return false
	}
	seenRaw, err := fsx.ReadFile(filepath.Join(dataDir, "crash.seen"))
	if err != nil {
		return false
	}
	seen, err := strconv.ParseInt(strings.TrimSpace(string(seenRaw)), 10, 64)
	if err != nil || seen < info.Size() {
		return false
	}
	if err := os.Truncate(path, 0); err != nil {
		return false
	}
	_ = datadir.WriteFile(filepath.Join(dataDir, "crash.seen"), []byte("0\n"), 0o600)
	return true
}
