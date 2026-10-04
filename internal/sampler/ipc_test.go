//go:build !windows

package sampler

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSessionUsesDiagnosticsSocket(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	pid := os.Getpid()
	socket := filepath.Join(temp, "dotnet-diagnostic-"+strconv.Itoa(pid)+"-test-socket")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	serverErr := make(chan error, 1)
	release := make(chan struct{})
	go func() {
		first, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		_, command, _, err := readFrame(first)
		if err == nil && command != collectTracing2 {
			err = errUnexpectedCommand
		}
		if err == nil {
			err = writeMessage(first, 0, appendUint64(nil, 41))
		}
		if err == nil {
			_, err = first.Write([]byte("Nettrace"))
		}
		if err != nil {
			_ = first.Close()
			serverErr <- err
			return
		}
		second, acceptErr := listener.Accept()
		if acceptErr != nil {
			_ = first.Close()
			serverErr <- acceptErr
			return
		}
		_, stopCommand, _, stopErr := readFrame(second)
		if stopErr == nil && stopCommand != stopTracing {
			stopErr = errUnexpectedCommand
		}
		if stopErr == nil {
			stopErr = writeMessage(second, 0, nil)
		}
		_ = second.Close()
		close(release)
		<-release
		_ = first.Close()
		serverErr <- stopErr
	}()

	session, err := Start(context.Background(), pid)
	if err != nil {
		t.Fatal(err)
	}
	path, err := session.Stop(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("session returned an empty trace path")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len("Nettrace")) {
		t.Fatalf("trace size = %d, want %d", info.Size(), len("Nettrace"))
	}
	_ = os.Remove(path)
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("fake diagnostics server did not finish")
	}
}

var errUnexpectedCommand = io.ErrUnexpectedEOF

func TestAnErrorReplyIsAnError(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, 20)
	copy(header, diagnosticMagic[:])
	binary.LittleEndian.PutUint16(header[14:16], 24)
	header[16], header[17] = serverCommandSet, serverError
	buf.Write(header)
	buf.Write([]byte{0x05, 0x00, 0x07, 0x80})
	if _, err := readMessage(&buf); err == nil || !strings.Contains(err.Error(), "0x80070005") {
		t.Fatalf("error reply read as %v", err)
	}
}
