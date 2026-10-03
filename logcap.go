package main

import (
	"io"
	"sync"
)

// maxLogBytes caps mortar.log so a long session cannot fill the disk; the previous run's log is kept separately.
const maxLogBytes = 10 << 20

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
