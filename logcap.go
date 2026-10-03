package main

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	if c.ended {
		return len(p), nil
	}
	if int64(len(p)) > c.left {
		c.ended = true
		_, err := io.WriteString(c.w, "mortar.log reached its size limit; later lines go to stderr only\n")
		return len(p), err
	}
	c.left -= int64(len(p))
	return c.w.Write(p)
}

func capCrashLog(dataDir string) {
	capCrashLogAt(dataDir, maxCrashBytes)
}

func capCrashLogAt(dataDir string, limit int64) {
	path := filepath.Join(dataDir, "crash.log")
	info, err := os.Stat(path)
	if err != nil || info.Size() <= limit {
		return
	}
	seenRaw, err := os.ReadFile(filepath.Join(dataDir, "crash.seen"))
	if err != nil {
		return
	}
	seen, err := strconv.ParseInt(strings.TrimSpace(string(seenRaw)), 10, 64)
	if err != nil || seen < info.Size() {
		return
	}
	if err := os.Truncate(path, 0); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dataDir, "crash.seen"), []byte("0\n"), 0o600)
}
