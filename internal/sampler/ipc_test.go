//go:build !windows

package sampler

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestSessionUsesDiagnosticsSocket(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	pid := os.Getpid()
	socket := filepath.Join(temp, "dotnet-diagnostic-"+strconv.Itoa(pid)+"-test-socket")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	release := make(chan struct{})
	go func() {
		first, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		command, _, err := readMessage(first)
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
		stopCommand, _, stopErr := readMessage(second)
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
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Nettrace" {
		t.Fatalf("trace = %q, want stream bytes", data)
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
