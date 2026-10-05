package bridge

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func listen(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is %T", ln.Addr())
	}
	return ln, addr.Port
}

// serve answers one connection per call with reply after reading the token and command lines, which it sends on got.
func serve(t *testing.T, reply string, hang bool) (port int, got chan string) {
	t.Helper()
	ln, port := listen(t)
	t.Cleanup(func() { _ = ln.Close() })
	got = make(chan string, 1)
	hold := 2 * timeout
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		r := bufio.NewReader(c)
		token, _ := r.ReadString('\n')
		cmd, _ := r.ReadString('\n')
		got <- token + cmd
		if hang {
			time.Sleep(hold)
			return
		}
		_, _ = fmt.Fprint(c, reply)
	}()
	return port, got
}

func state(port int) State { return State{Port: port, Token: "tok", PID: os.Getpid()} }

func TestSendOK(t *testing.T) {
	port, got := serve(t, "ok\n", false)
	if err := send(t.Context(), state(port), `help "a b"`); err != nil {
		t.Fatal(err)
	}
	if line := <-got; line != "tok\nhelp \"a b\"\n" {
		t.Fatalf("server saw %q", line)
	}
}

func TestSendRejected(t *testing.T) {
	port, _ := serve(t, "error: missing or oversized command\n", false)
	var re *RejectedError
	if err := send(t.Context(), state(port), "x"); !errors.As(err, &re) || re.Message != "missing or oversized command" {
		t.Fatalf("err = %v", err)
	}
}

func TestSendWrongToken(t *testing.T) {
	port, _ := serve(t, "error: unauthorized\n", false)
	if err := send(t.Context(), state(port), "x"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestSendTimeout(t *testing.T) {
	timeout = 200 * time.Millisecond
	t.Cleanup(func() { timeout = 3 * time.Second })
	port, _ := serve(t, "", true)
	start := time.Now()
	if err := send(t.Context(), state(port), "x"); err == nil || time.Since(start) > 2*timeout {
		t.Fatalf("err = %v after %v", err, time.Since(start))
	}
}

func TestSendNothingListening(t *testing.T) {
	ln, port := listen(t)
	_ = ln.Close()
	if err := send(t.Context(), state(port), "x"); !errors.Is(err, ErrNotReady) {
		t.Fatalf("err = %v", err)
	}
}

func TestMultilineCommandRefusedBeforeConnecting(t *testing.T) {
	var re *RejectedError
	if err := send(t.Context(), state(1), "a\nb"); !errors.As(err, &re) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadState(t *testing.T) {
	write := func(dir, body string) {
		if err := os.WriteFile(filepath.Join(dir, StateFile), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	live := fmt.Sprintf(`{"port":51234,"token":"abc","pid":%d}`, os.Getpid())
	cases := []struct {
		name, body string
		want       error
	}{
		{"missing", "", ErrNotReady},
		{"garbage", "{", ErrNotReady},
		{"no token", fmt.Sprintf(`{"port":51234,"token":"","pid":%d}`, os.Getpid()), ErrNotReady},
		{"dead pid", `{"port":51234,"token":"abc","pid":2147483646}`, ErrNotRunning},
		{"live", live, nil},
	}
	for _, c := range cases {
		dir := t.TempDir()
		if c.body != "" {
			write(dir, c.body)
		}
		st, err := ReadState(dir)
		if !errors.Is(err, c.want) {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
		if c.want == nil && (st.Port != 51234 || st.Token != "abc") {
			t.Fatalf("%s: state = %+v", c.name, st)
		}
	}
}
