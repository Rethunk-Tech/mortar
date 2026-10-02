// Package control lets the mortar command line drive the running app. The app listens on 127.0.0.1 and writes the
// port and a random token to control.json in its data folder (readable by the user only); each connection carries
// one JSON request line and gets one JSON reply line. Only a process that can read that file can call it.
package control

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

// FileName is the discovery file the app writes in its data folder while it runs.
const FileName = "control.json"

// maxLine bounds a request or reply line.
const maxLine = 16 << 20

// Params is every argument a method takes; each method reads the fields it needs.
type Params struct {
	Game      string   `json:"game,omitempty"`
	Profile   string   `json:"profile,omitempty"`
	Name      string   `json:"name,omitempty"`
	UniqueIDs []string `json:"uniqueIds,omitempty"`
	Path      string   `json:"path,omitempty"`
	Run       string   `json:"run,omitempty"`
	All       bool     `json:"all,omitempty"`
	Unused    bool     `json:"unused,omitempty"`
	Yes       bool     `json:"yes,omitempty"`
}

type request struct {
	Token  string `json:"token"`
	Method string `json:"method"`
	Params Params `json:"params"`
}

type reply struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type discovery struct {
	Port    int    `json:"port"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
	Version string `json:"version"`
}

// Handler runs one method.
type Handler func(ctx context.Context, method string, p Params) (any, error)

// Serve listens until ctx ends, then removes the discovery file.
func Serve(ctx context.Context, dir, version string, h Handler) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	tok := make([]byte, 32)
	if _, err := rand.Read(tok); err != nil {
		_ = ln.Close()
		return err
	}
	token := hex.EncodeToString(tok)
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return errors.New("control: listener has no TCP address")
	}
	path := filepath.Join(dir, FileName)
	b, err := json.Marshal(discovery{Port: tcp.Port, Token: token, PID: os.Getpid(), Version: version})
	if err != nil {
		_ = ln.Close()
		return err
	}
	if err := datadir.WriteFile(path, b, 0o600); err != nil {
		_ = ln.Close()
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
		_ = os.Remove(path)
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		go serveConn(ctx, conn, token, h)
	}
}

func serveConn(ctx context.Context, conn net.Conn, token string, h Handler) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReaderSize(conn, 64<<10)
	var req request
	if err := json.NewDecoder(io.LimitReader(r, maxLine)).Decode(&req); err != nil {
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(token)) != 1 {
		writeReply(conn, reply{Error: "unauthorized"})
		return
	}
	log.Printf("control: %s %s %s", req.Method, req.Params.Game, req.Params.Profile)
	res, err := h(ctx, req.Method, req.Params)
	if err != nil {
		writeReply(conn, reply{Error: err.Error()})
		return
	}
	b, err := json.Marshal(res)
	if err != nil {
		writeReply(conn, reply{Error: err.Error()})
		return
	}
	writeReply(conn, reply{Result: b})
}

func writeReply(w io.Writer, rep reply) {
	b, err := json.Marshal(rep)
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}

// ErrNotRunning means no running Mortar answered.
var ErrNotRunning = errors.New("the Mortar app is not running: start it, then run this command again")

// Call sends one request to the running app and decodes its result into out (nil to discard it). timeout bounds
// the whole call; methods that wait on the game take longer than reads.
func Call(method string, p Params, out any, timeout time.Duration) error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	return CallDir(dir, method, p, out, timeout)
}

// CallDir is Call against the app whose data folder is dir.
func CallDir(dir, method string, p Params, out any, timeout time.Duration) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotRunning
		}
		return err
	}
	b, err := root.ReadFile(FileName)
	_ = root.Close()
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotRunning
		}
		return err
	}
	var d discovery
	if err := json.Unmarshal(b, &d); err != nil {
		return fmt.Errorf("control: %s: %w", FileName, err)
	}
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", d.Port))
	if err != nil {
		return ErrNotRunning
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	req, err := json.Marshal(request{Token: d.Token, Method: method, Params: p})
	if err != nil {
		return err
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return err
	}
	var rep reply
	if err := json.NewDecoder(io.LimitReader(conn, maxLine)).Decode(&rep); err != nil {
		return fmt.Errorf("control: reading the reply: %w", err)
	}
	if rep.Error != "" {
		return errors.New(rep.Error)
	}
	if out == nil || len(rep.Result) == 0 {
		return nil
	}
	return json.Unmarshal(rep.Result, out)
}
