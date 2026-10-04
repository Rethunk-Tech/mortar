package sampler

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
)

const (
	diagnosticCommandSet = 0x02
	collectTracing2      = 0x03
	stopTracing          = 0x01
)

var diagnosticMagic = [14]byte{'D', 'O', 'T', 'N', 'E', 'T', '_', 'I', 'P', 'C', '_', 'V', '1'}

type diagnosticConn interface {
	io.Reader
	io.Writer
	io.Closer
}

type diagnosticConnection struct {
	diagnosticConn
}

// Session owns one EventPipe stream and its temporary nettrace file.
type Session struct {
	pid       int
	conn      diagnosticConn
	path      string
	sessionID uint64
	streamErr chan error
	done      chan struct{}
	stopOnce  sync.Once
	stopErr   error
	mu        sync.Mutex
}

// Start connects to a process's diagnostics endpoint and starts a nettrace
// stream with the sample profiler and runtime JIT/loader providers enabled.
func Start(ctx context.Context, pid int) (*Session, error) {
	conn, err := dialDiagnostic(ctx, pid)
	if err != nil {
		return nil, err
	}
	if err := writeMessage(conn, collectTracing2, collectPayload()); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_, payload, err := readMessage(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if len(payload) < 8 {
		_ = conn.Close()
		return nil, errors.New("sampler: diagnostics response has no session id")
	}
	file, err := os.CreateTemp("", "mortar-startup-*.nettrace")
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	session := &Session{
		pid:       pid,
		conn:      conn,
		path:      file.Name(),
		sessionID: binary.LittleEndian.Uint64(payload[:8]),
		streamErr: make(chan error, 1),
		done:      make(chan struct{}),
	}
	go session.copyStream(file)
	return session, nil
}

// Path returns the temporary nettrace path. It is valid after Wait returns.
func (s *Session) Path() string {
	return s.path
}

// Wait waits for the stream connection to close and returns its copy error.
func (s *Session) Wait() error {
	return <-s.streamErr
}

// Stop requests rundown, waits for the stream to finish, and returns the
// completed nettrace path.
func (s *Session) Stop(ctx context.Context) (string, error) {
	s.stopOnce.Do(func() {
		s.stopErr = s.stop(ctx)
	})
	<-s.done
	s.mu.Lock()
	err := errors.Join(s.stopErr, <-s.streamErr)
	s.mu.Unlock()
	return s.path, err
}

func (s *Session) stop(ctx context.Context) error {
	conn, err := dialDiagnostic(ctx, s.pid)
	if err != nil {
		_ = s.conn.Close()
		return err
	}
	defer func() {
		_ = conn.Close()
	}()
	if err := writeMessage(conn, stopTracing, stopPayload(s.sessionID)); err != nil {
		_ = s.conn.Close()
		return err
	}
	_, _, err = readMessage(conn)
	if err != nil {
		_ = s.conn.Close()
		return err
	}
	return nil
}

func (s *Session) copyStream(file *os.File) {
	_, copyErr := io.Copy(file, s.conn)
	if syncErr := file.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := file.Close(); copyErr == nil {
		copyErr = closeErr
	}
	_ = s.conn.Close()
	s.streamErr <- copyErr
	close(s.done)
}

func collectPayload() []byte {
	var payload []byte
	payload = appendUint32(payload, 256)
	payload = appendUint32(payload, 1)
	payload = append(payload, 1)
	payload = appendUint32(payload, 2)
	payload = appendProvider(payload, 0, 5, "Microsoft-DotNETCore-SampleProfiler")
	payload = appendProvider(payload, 0x18, 5, "Microsoft-Windows-DotNETRuntime")
	return payload
}

func appendProvider(payload []byte, keywords uint64, level uint32, name string) []byte {
	payload = appendUint64(payload, keywords)
	payload = appendUint32(payload, level)
	payload = appendWideString(payload, name)
	return appendWideString(payload, "")
}

func stopPayload(sessionID uint64) []byte {
	return appendUint64(nil, sessionID)
}

func appendUint32(payload []byte, value uint32) []byte {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	return append(payload, encoded[:]...)
}

func appendUint64(payload []byte, value uint64) []byte {
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], value)
	return append(payload, encoded[:]...)
}

func appendWideString(payload []byte, value string) []byte {
	encoded := utf16.Encode([]rune(value))
	payload = appendUint32(payload, uint32Length(len(encoded)+1))
	for _, unit := range encoded {
		payload = append(payload, byte(unit&0xff), byte((unit>>8)&0xff))
	}
	return append(payload, 0, 0)
}

func uint32Length(value int) uint32 {
	if value <= 0 {
		return 0
	}
	const maxUint32 = int(^uint32(0))
	if value > maxUint32 {
		return ^uint32(0)
	}
	return uint32(value)
}

func writeMessage(conn io.Writer, commandID byte, payload []byte) error {
	size := 20 + len(payload)
	if size > 0xffff {
		return errors.New("sampler: diagnostics message is too large")
	}
	header := make([]byte, 20)
	copy(header[:14], diagnosticMagic[:])
	binary.LittleEndian.PutUint16(header[14:16], uint16(size))
	header[16] = diagnosticCommandSet
	header[17] = commandID
	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

func readMessage(conn io.Reader) (byte, []byte, error) {
	header := make([]byte, 20)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}
	if string(header[:14]) != string(diagnosticMagic[:]) {
		return 0, nil, errors.New("sampler: invalid diagnostics response magic")
	}
	size := int(binary.LittleEndian.Uint16(header[14:16]))
	if size < 20 {
		return 0, nil, errors.New("sampler: invalid diagnostics response size")
	}
	payload := make([]byte, size-20)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}
	return header[17], payload, nil
}

func diagnosticSocketCandidates(pid int) ([]string, error) {
	if pid <= 0 {
		return nil, errors.New("sampler: invalid process id")
	}
	if runtime.GOOS == "windows" {
		return []string{fmt.Sprintf(`\\.\pipe\dotnet-diagnostic-%d`, pid)}, nil
	}
	dirs := []string{os.TempDir()}
	dirs = append(dirs, filepath.Join("/proc", fmt.Sprint(pid), "root", "tmp"))
	if env, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "environ")); err == nil {
		for entry := range strings.SplitSeq(string(env), "\x00") {
			if value, ok := strings.CutPrefix(entry, "TMPDIR="); ok && value != "" {
				dirs = append(dirs, value)
			}
		}
	}
	seen := make(map[string]bool)
	var matches []string
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		found, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("dotnet-diagnostic-%d-*-socket", pid)))
		if err != nil {
			return nil, err
		}
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, os.ErrNotExist
	}
	return matches, nil
}

func dialDiagnostic(ctx context.Context, pid int) (*diagnosticConnection, error) {
	if ctx == nil {
		return nil, errors.New("sampler: nil diagnostics context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	paths, err := diagnosticSocketCandidates(pid)
	if err != nil {
		return nil, err
	}
	var last error
	for _, path := range paths {
		conn, err := openDiagnostic(ctx, path)
		if err == nil {
			return &diagnosticConnection{diagnosticConn: conn}, nil
		}
		last = err
	}
	if last == nil {
		last = os.ErrNotExist
	}
	return nil, last
}
