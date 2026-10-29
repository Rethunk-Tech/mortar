//go:build unix

package avscan

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestClamdSkipsDefaultSocketOwnedByAnotherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root owns the socket")
	}
	path := filepath.Join(t.TempDir(), "clamd.sock")
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	old := defaultSockets
	defaultSockets = []string{path}
	defer func() { defaultSockets = old }()
	if got := (clamd{}).candidates(); len(got) != 0 {
		t.Fatalf("a socket owned by this user is trusted: %v", got)
	}
	if got := (clamd{socket: path}).candidates(); len(got) != 1 {
		t.Fatalf("a configured socket is trusted as given: %v", got)
	}
}
