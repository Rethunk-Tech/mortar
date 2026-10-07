package avscan

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// eicar is the standard antivirus test string, assembled at run time so no file in the repo holds it whole.
func eicar() string {
	return strings.Join([]string{`X5O!P%@AP[4\PZX54(P^)7CC)7}$`, "EICAR-STANDARD-ANTIVIRUS-TEST-FILE", `!$H+H*`}, "")
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fakeClamd listens on a unix socket and answers like clamd: INSTREAM chunks are read and the reply is FOUND when the
// stream holds the EICAR string, or an ERROR once more than limit bytes arrive.
func fakeClamd(t *testing.T, limit int) clamd {
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
			go serveClamd(conn, limit)
		}
	}()
	return clamd{socket: path}
}

func serveClamd(conn net.Conn, limit int) {
	defer func() { _ = conn.Close() }()
	var cmd []byte
	one := make([]byte, 1)
	for {
		if _, err := io.ReadFull(conn, one); err != nil {
			return
		}
		if one[0] == 0 {
			break
		}
		cmd = append(cmd, one[0])
	}
	if string(cmd) == "zVERSION" {
		_, _ = conn.Write([]byte("ClamAV 1.0.1/27000/Mon\x00"))
		return
	}
	var got []byte
	for {
		var size [4]byte
		if _, err := io.ReadFull(conn, size[:]); err != nil {
			return
		}
		n := binary.BigEndian.Uint32(size[:])
		if n == 0 {
			break
		}
		chunk := make([]byte, n)
		if _, err := io.ReadFull(conn, chunk); err != nil {
			return
		}
		got = append(got, chunk...)
		if limit > 0 && len(got) > limit {
			_, _ = conn.Write([]byte("INSTREAM size limit exceeded. ERROR\x00"))
			return
		}
	}
	if strings.Contains(string(got), eicar()) {
		_, _ = conn.Write([]byte("stream: Eicar-Test-Signature FOUND\x00"))
		return
	}
	_, _ = conn.Write([]byte("stream: OK\x00"))
}

func TestClamdProtocolFlagsOnlyTheInfectedFile(t *testing.T) {
	t.Parallel()
	dir := writeTree(t, map[string]string{"a/clean.dll": "hello", "b/bad.dll": "prefix" + eicar()})
	got, found, err := fakeClamd(t, 0).Scan(context.Background(), dir)
	if err != nil || !found || got.Name != "Eicar-Test-Signature" || got.File != "b/bad.dll" || got.Scanner != "clamd" {
		t.Fatalf("detection = %+v, err = %v", got, err)
	}
	clean := writeTree(t, map[string]string{"a.dll": "fine"})
	if got, found, err := fakeClamd(t, 0).Scan(context.Background(), clean); found || err != nil {
		t.Fatalf("clean tree = %+v, %v", got, err)
	}
}

func TestClamdStreamLimitIsSkippedNotAnError(t *testing.T) {
	t.Parallel()
	dir := writeTree(t, map[string]string{"big.bin": strings.Repeat("x", 200_000)})
	if got, found, err := fakeClamd(t, 100_000).Scan(context.Background(), dir); found || err != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestClamdWithoutADaemonIsNoScanner(t *testing.T) {
	t.Parallel()
	dir := writeTree(t, map[string]string{"a.dll": "x"})
	c := clamd{socket: filepath.Join(t.TempDir(), "none.sock")}
	if _, _, err := c.Scan(context.Background(), dir); !errors.Is(err, ErrNoScanner) {
		t.Fatalf("err = %v", err)
	}
}

func TestClamdVersionNamesTheDaemon(t *testing.T) {
	t.Parallel()
	if got := clamdProduct(context.Background(), fakeClamd(t, 0)); got != "clamd 1.0.1" {
		t.Fatalf("product = %q", got)
	}
	if got := clamdProduct(context.Background(), clamd{socket: filepath.Join(t.TempDir(), "none.sock")}); got != "No antivirus found" {
		t.Fatalf("product = %q", got)
	}
}

func TestSplitCommandKeepsQuotedParts(t *testing.T) {
	t.Parallel()
	got := splitCommand(`"C:\Program Files\av\scan.exe" --quiet "{path}"`)
	if len(got) != 3 || got[0] != `C:\Program Files\av\scan.exe` || got[2] != "{path}" {
		t.Fatalf("parts = %q", got)
	}
}

func TestCustomCommandExitCodeDecidesAndStdoutNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	t.Parallel()
	dir := writeTree(t, map[string]string{"a.dll": "x"})
	if got, found, err := (command{line: `sh -c "test -d {path}"`}).Scan(context.Background(), dir); found || err != nil {
		t.Fatalf("clean = %+v, %v", got, err)
	}
	got, found, err := command{line: `sh -c "echo Win.Test.Bad; exit 1" {path}`}.Scan(context.Background(), dir)
	if err != nil || !found || got.Name != "Win.Test.Bad" || got.Scanner != "sh" {
		t.Fatalf("detection = %+v, %v", got, err)
	}
	if _, _, err := (command{line: "no-such-scanner-xyz {path}"}).Scan(context.Background(), dir); err == nil || errors.Is(err, ErrNoScanner) {
		t.Fatalf("a missing program is an error, got %v", err)
	}
	if _, _, err := (command{}).Scan(context.Background(), dir); !errors.Is(err, ErrNoScanner) {
		t.Fatalf("empty command err = %v", err)
	}
}

func TestOffScansNothing(t *testing.T) {
	t.Parallel()
	dir := writeTree(t, map[string]string{"bad": eicar()})
	if got, found, err := New(Config{Mode: ModeOff}).Scan(context.Background(), dir); found || err != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}
