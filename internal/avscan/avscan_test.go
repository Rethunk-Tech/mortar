package avscan

import (
	"context"
	"net"
	"path/filepath"
	"runtime"
	"testing"

	goavscan "github.com/Rethunk-Tech/go-avscan"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// fakeClamd answers zVERSION like clamd and nothing else.
func fakeClamd(t *testing.T) goavscan.Clamd {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clamd.sock")
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", path)
	if err != nil {
		t.Skipf("no unix sockets: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write([]byte("ClamAV 1.0.1/27000/Mon\x00"))
			_ = conn.Close()
		}
	}()
	return goavscan.Clamd{Socket: path}
}

func TestClamdVersionNamesTheDaemon(t *testing.T) {
	t.Parallel()
	if got := clamdProduct(context.Background(), fakeClamd(t)); got.Text != "clamd 1.0.1" || got.Code != "" {
		t.Fatalf("product = %+v", got)
	}
	if got := clamdProduct(context.Background(), goavscan.Clamd{Socket: filepath.Join(t.TempDir(), "none.sock")}); got.Text != "No antivirus found" || got.Code != "none" {
		t.Fatalf("product = %+v", got)
	}
}

func TestOffScansNothing(t *testing.T) {
	t.Parallel()
	if got, found, err := New(Config{Mode: ModeOff}).Scan(context.Background(), t.TempDir()); found || err != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestACommandThatPrintsNothingIsNamedAfterItsProgram(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("needs a unix false")
	}
	dir := t.TempDir()
	testfs.WriteAt(t, filepath.Join(dir, "a"), []byte("x"))
	got, found, err := New(Config{Mode: ModeCommand, Command: "false"}).Scan(context.Background(), dir)
	if err != nil || !found || got.Name != "detected by false" || got.Scanner != "false" {
		t.Fatalf("got %+v, %v, %v", got, found, err)
	}
}

func TestStatusSaysWhetherTheScannerCanRun(t *testing.T) {
	t.Parallel()
	if st := StatusOf(t.Context(), Config{Mode: ModeOff}); !st.Ready || st.Scanner != "off" || st.Mode != ModeOff {
		t.Fatalf("off = %+v", st)
	}
	st := StatusOf(t.Context(), Config{Mode: ModeCommand})
	if st.Ready || st.Problem == "" || st.Scanner != "command" {
		t.Fatalf("a command mode with no command is not ready: %+v", st)
	}
	if st := StatusOf(t.Context(), Config{Mode: ModeClamd, Socket: filepath.Join(t.TempDir(), "none.sock")}); st.Ready || st.Problem == "" {
		t.Fatalf("clamd without a daemon is not ready: %+v", st)
	}
}
