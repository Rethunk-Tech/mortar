package controlwire

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnansweringNamesAFrozenAppOnly(t *testing.T) {
	dir := t.TempDir()
	if _, ok := Unanswering(dir, time.Second); ok {
		t.Fatal("no control file counted as frozen")
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	// Hang up on hello rather than leave it to its timeout: either way the app did not answer.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v", ln.Addr())
	}
	b, err := json.Marshal(Discovery{Port: addr.Port, Token: "t", PID: 4242, Protocol: Protocol})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if pid, ok := Unanswering(dir, 0); !ok || pid != 4242 {
		t.Fatalf("frozen app = %d, %v", pid, ok)
	}
}
