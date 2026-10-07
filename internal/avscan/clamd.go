package avscan

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// defaultSockets are where distributions put clamd's socket.
var defaultSockets = []string{
	"/run/clamav/clamd.ctl",
	"/var/run/clamav/clamd.ctl",
	"/run/clamd.scan/clamd.sock",
	"/var/run/clamd.scan/clamd.sock",
	"/tmp/clamd.socket",
}

const (
	clamdChunk   = 64 * 1024
	clamdTimeout = 30 * time.Second
)

// clamd streams each file to a clamd daemon over its socket (INSTREAM), so the daemon needs no read access to
// Mortar's folders.
type clamd struct {
	socket string
	// dial is for tests; the zero value dials a unix socket.
	dial func(ctx context.Context, path string) (net.Conn, error)
}

func (c clamd) candidates() []string {
	if c.socket != "" {
		return []string{c.socket}
	}
	return defaultSockets
}

func (c clamd) connect(ctx context.Context) (net.Conn, error) {
	dial := c.dial
	if dial == nil {
		dial = func(ctx context.Context, path string) (net.Conn, error) {
			return (&net.Dialer{Timeout: clamdTimeout}).DialContext(ctx, "unix", path)
		}
	}
	for _, path := range c.candidates() {
		if conn, err := dial(ctx, path); err == nil {
			return conn, nil
		}
	}
	return nil, ErrNoScanner
}

// Scan streams every file under dir and returns the first detection.
func (c clamd) Scan(ctx context.Context, dir string) (Detection, bool, error) {
	var found Detection
	var hit bool
	errStop := errors.New("stop")
	err := files(ctx, dir, func(abs, rel string) error {
		name, err := c.scanFile(ctx, abs)
		if err != nil {
			return err
		}
		if name != "" {
			found, hit = Detection{Name: name, File: rel, Scanner: "clamd"}, true
			return errStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return Detection{}, false, err
	}
	return found, hit, nil
}

// scanFile returns the threat clamd names for the file, or "" when it is clean or too large for clamd's stream limit.
func (c clamd) scanFile(ctx context.Context, path string) (string, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	deadline := time.Now().Add(clamdTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline.Add(clamdTimeout)) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	// A cancelled scan must not sit out a blocked read or write until the deadline.
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()
	in, err := fsx.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return "", err
	}
	buf := make([]byte, clamdChunk)
	size := make([]byte, 4)
	for {
		n, readErr := in.Read(buf)
		if n > 0 {
			if n > math.MaxUint32 {
				return "", errors.New("chunk too large")
			}
			binary.BigEndian.PutUint32(size, uint32(n))
			if _, err := conn.Write(size); err != nil {
				return replyOnWriteError(conn, err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return replyOnWriteError(conn, err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return "", err
	}
	return parseReply(conn)
}

// replyOnWriteError reads what clamd said when it closed the stream early (its size limit), else reports the write
// error.
func replyOnWriteError(conn net.Conn, writeErr error) (string, error) {
	name, err := parseReply(conn)
	if err == nil {
		return name, nil
	}
	return "", writeErr
}

// parseReply reads one NUL-terminated answer: "stream: OK", "stream: <name> FOUND" or "<message> ERROR".
func parseReply(r io.Reader) (string, error) {
	line, err := bufio.NewReader(io.LimitReader(r, 4096)).ReadString(0)
	if err != nil && line == "" {
		return "", err
	}
	line = strings.TrimSpace(strings.TrimRight(line, "\x00"))
	switch {
	case strings.HasSuffix(line, " FOUND"):
		name := strings.TrimSuffix(line, " FOUND")
		return strings.TrimSpace(strings.TrimPrefix(name, "stream:")), nil
	case strings.HasSuffix(line, "size limit exceeded. ERROR"):
		return "", nil
	case strings.HasSuffix(line, " OK"):
		return "", nil
	}
	return "", fmt.Errorf("clamd answered %q", line)
}

// Version asks clamd for its version line ("ClamAV 1.0.1/..."), reporting ErrNoScanner when no daemon answers.
func (c clamd) Version(ctx context.Context) (string, error) {
	conn, err := c.connect(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(clamdTimeout))
	if _, err := conn.Write([]byte("zVERSION\x00")); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(io.LimitReader(conn, 4096)).ReadString(0)
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(strings.TrimRight(line, "\x00")), nil
}
