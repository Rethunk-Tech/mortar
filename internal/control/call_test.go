package control

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
)

// fakeApp answers on the data folder's control file, hello with protocol, every other method "ok", and reports the
// method of each request it reads in order.
func fakeApp(t *testing.T, protocol int) (seen chan string) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	seen = make(chan string, 16)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				line, err := bufio.NewReader(c).ReadBytes('\n')
				if err != nil {
					return
				}
				var req struct {
					Method string `json:"method"`
				}
				if err := json.Unmarshal(line, &req); err != nil {
					return
				}
				seen <- req.Method
				if req.Method == "hello" {
					_, _ = fmt.Fprintf(c, `{"result":{"version":"9.9.9","protocol":%d}}`+"\n", protocol)
					return
				}
				_, _ = fmt.Fprint(c, "{\"result\":\"ok\"}\n")
			}(conn)
		}
	}()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("no TCP address")
	}
	d, _ := json.Marshal(controlwire.Discovery{Port: tcp.Port, Token: "t", Protocol: protocol})
	if err := os.WriteFile(filepath.Join(dir, controlwire.FileName), d, 0o600); err != nil {
		t.Fatal(err)
	}
	return seen
}

func TestCallRefusesAnotherProtocolAndChecksAgainNextTime(t *testing.T) {
	protocolChecked.Store(false)
	t.Cleanup(func() { protocolChecked.Store(false) })
	seen := fakeApp(t, controlwire.Protocol+1)
	want := fmt.Sprintf("this mortar CLI speaks control protocol %d, the app speaks %d", controlwire.Protocol, controlwire.Protocol+1)
	for range 2 {
		if err := Call("hello", Params{}, nil, time.Second); err == nil || err.Error() != want {
			t.Fatalf("err = %v, want %q", err, want)
		}
	}
	if got := len(seen); got != 2 {
		t.Fatalf("hello went out %d times, want 2: a refused protocol must not be remembered", got)
	}
}

func TestCallChecksTheProtocolOnceAfterAMatch(t *testing.T) {
	protocolChecked.Store(false)
	t.Cleanup(func() { protocolChecked.Store(false) })
	seen := fakeApp(t, controlwire.Protocol)
	for range 2 {
		var got string
		if err := Call("echo", Params{}, &got, time.Second); err != nil || got != "ok" {
			t.Fatalf("call = %q, %v", got, err)
		}
	}
	var methods []string
	for len(seen) > 0 {
		methods = append(methods, <-seen)
	}
	if len(methods) != 3 || methods[0] != "hello" || methods[1] != "echo" || methods[2] != "echo" {
		t.Fatalf("requests = %v, want one hello then the two calls", methods)
	}
}
