package avscan

import (
	"errors"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	goavscan "github.com/Totative/go-avscan"
)

// fakeClamd answers zVERSION like clamd and nothing else.
func fakeClamd(t *testing.T) goavscan.Clamd {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clamd.sock")
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", path)
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
			// Reading the command first keeps the close from resetting the connection before the client has read.
			_, _ = conn.Read(make([]byte, 64))
			_, _ = conn.Write([]byte("ClamAV 1.0.1/27000/Mon\x00"))
			_ = conn.Close()
		}
	}()
	return goavscan.Clamd{Socket: path}
}

func TestClamdVersionNamesTheDaemon(t *testing.T) {
	t.Parallel()
	if got := clamdProduct(t.Context(), fakeClamd(t)); got.Text != "clamd 1.0.1" || got.Code != "" {
		t.Fatalf("product = %+v", got)
	}
	if got := clamdProduct(t.Context(), goavscan.Clamd{Socket: filepath.Join(t.TempDir(), "none.sock")}); got.Text != "No antivirus found" || got.Code != "none" {
		t.Fatalf("product = %+v", got)
	}
}

func TestOffScansNothing(t *testing.T) {
	t.Parallel()
	if got, found, err := New(Config{Mode: ModeOff}).Scan(t.Context(), t.TempDir()); found || err != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestOnlyAnUnnamedCommandDetectionIsNamedAfterItsProgram(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("needs a unix sh")
	}
	dir := t.TempDir()
	testfs.WriteAt(t, filepath.Join(dir, "a"), []byte("x"))
	got, found, err := New(Config{Mode: ModeCommand, Command: "false {path}"}).Scan(t.Context(), dir)
	if err != nil || !found || got.Name != "detected by false" {
		t.Errorf("unnamed: got %+v, %v, %v", got, found, err)
	}
	got, found, err = New(Config{Mode: ModeCommand, Command: `sh -c "echo Win.Test.Bad; exit 1" {path}`}).Scan(t.Context(), dir)
	if err != nil || !found || got.Name != "Win.Test.Bad" {
		t.Errorf("named: got %+v, %v, %v", got, found, err)
	}
	got, found, err = New(Config{Mode: ModeCommand, Command: "true {path}"}).Scan(t.Context(), dir)
	if err != nil || found || got != (goavscan.Detection{}) {
		t.Errorf("clean: got %+v, %v, %v", got, found, err)
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

func TestAScannerThePlayerSetUpThatCannotBeReachedIsReported(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		cfg  Config
		tell bool
		want string
	}{
		"nothing set up": {Config{}, false, ""},
		"automatic only": {Config{Mode: ModeAutomatic}, false, ""},
		"clamd socket":   {Config{Mode: ModeClamd, Socket: "/run/clamd.sock"}, true, "/run/clamd.sock"},
		"socket in auto": {Config{Socket: "/run/clamd.sock"}, true, "/run/clamd.sock"},
		"custom command": {Config{Mode: ModeCommand, Command: "scan {path}"}, true, "scan {path}"},
		"other failure":  {Config{}, true, "boom"},
	} {
		err := goavscan.ErrNoScanner
		if name == "other failure" {
			err = errors.New("boom")
		}
		got, tell := c.cfg.Notice(err)
		if tell != c.tell || !strings.Contains(got, c.want) {
			t.Errorf("%s: notice %q tell %v, want %q %v", name, got, tell, c.want, c.tell)
		}
	}
}
