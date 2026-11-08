package bridgetest

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
)

// token is 64 lowercase hex so ReadState accepts it; a shorter or mixed-case
// value fails as ErrNotReady.
var token = strings.Repeat("a", 64)

// FakeBridge starts a loopback companion listener, writes a valid state file
// under root via c.StatePath, and returns a channel of received commands.
// Each replies entry is written back as one line; extra commands get an empty line.
func FakeBridge(t *testing.T, c bridge.Companion, root string, replies ...string) <-chan string {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", ln.Addr())
	}

	cmds := make(chan string, 8)
	go func() {
		defer close(cmds)
		i := 0
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			func() {
				defer conn.Close()
				r := bufio.NewReader(conn)
				got, _ := r.ReadString('\n')
				if strings.TrimSpace(got) != token {
					return
				}
				cmd, _ := r.ReadString('\n')
				cmds <- strings.TrimSpace(cmd)
				reply := ""
				if i < len(replies) {
					reply = replies[i]
					i++
				}
				_, _ = fmt.Fprintln(conn, reply)
			}()
		}
	}()

	path := c.StatePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	state := fmt.Sprintf(`{"port":%d,"token":"%s"}`, addr.Port, token)
	if err := os.WriteFile(path, []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	return cmds
}
