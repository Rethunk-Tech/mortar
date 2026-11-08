package controlwire

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// liveServer answers hello on every accepted connection and counts the connections a probe opens.
func liveServer(t *testing.T, token string) (dir string, accepts *atomic.Int32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	dir = t.TempDir()
	b, err := json.Marshal(Discovery{Port: ln.Addr().(*net.TCPAddr).Port, Token: token, PID: 4242, Version: "test", Protocol: Protocol})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	accepts = new(atomic.Int32)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			accepts.Add(1)
			go func() {
				defer func() { _ = conn.Close() }()
				line, err := bufio.NewReader(conn).ReadBytes('\n')
				if err != nil {
					return
				}
				var req struct {
					Token  string `json:"token"`
					Method string `json:"method"`
				}
				if json.Unmarshal(line, &req) != nil || req.Token != token || req.Method != "hello" {
					return
				}
				_, _ = conn.Write([]byte(`{"result":{"version":"test","protocol":1}}` + "\n"))
			}()
		}
	}()
	return dir, accepts
}

func TestLiveAnswersOnOneConnection(t *testing.T) {
	dir, accepts := liveServer(t, "tok")
	pid, ok := Live(t.Context(), dir)
	if !ok || pid != 4242 {
		t.Fatalf("Live = %d, %v, want 4242, true", pid, ok)
	}
	if n := accepts.Load(); n != 1 {
		t.Fatalf("Live opened %d connections, want 1", n)
	}
}
