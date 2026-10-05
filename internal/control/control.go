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
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Params is every argument a method takes; each method reads the fields it needs.
type Params struct {
	Game    string `json:"game,omitempty"`
	Profile string `json:"profile,omitempty"`
	// Source is a mod source id (nexus, github, thunderstore); ID is a mod's id in that source (Nexus mod id, GitHub
	// owner/repo).
	Source string `json:"source,omitempty"`
	ID     string `json:"id,omitempty"`
	// Install is a game install id; empty means the game's only or default install.
	Install string `json:"install,omitempty"`
	// Page is a 1-based result page, Index a 1-based row in a listing.
	Page   int      `json:"page,omitempty"`
	Index  int      `json:"index,omitempty"`
	Name   string   `json:"name,omitempty"`
	IDs    []string `json:"ids,omitempty"`
	Path   string   `json:"path,omitempty"`
	Run    string   `json:"run,omitempty"`
	Query  string   `json:"query,omitempty"`
	All    bool     `json:"all,omitempty"`
	Unused bool     `json:"unused,omitempty"`
	Yes    bool     `json:"yes,omitempty"`
	Sub    string   `json:"sub,omitempty"`
	Force  bool     `json:"force,omitempty"`
	Key    string   `json:"key,omitempty"`
	Value  string   `json:"value,omitempty"`
	Remove bool     `json:"remove,omitempty"`
	Set    bool     `json:"set,omitempty"`
	Clear  bool     `json:"clear,omitempty"`
	Unlink bool     `json:"unlink,omitempty"`
	Keep   int      `json:"keep,omitempty"`
	Preset string   `json:"preset,omitempty"`
	// Loader names one of the game's loaders; empty means the catalog's first.
	Loader string `json:"loader,omitempty"`
}

type request struct {
	Token  string `json:"token"`
	Method string `json:"method"`
	Params Params `json:"params"`
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
	path := filepath.Join(dir, controlwire.FileName)
	b, err := json.Marshal(controlwire.Discovery{Port: tcp.Port, Token: token, PID: os.Getpid(), Version: version, Protocol: controlwire.Protocol})
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
		go serveConn(ctx, conn, token, version, h)
	}
}

func serveConn(ctx context.Context, conn net.Conn, token, version string, h Handler) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReaderSize(conn, 64<<10)
	var req request
	if err := json.NewDecoder(io.LimitReader(r, controlwire.MaxLine)).Decode(&req); err != nil {
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(token)) != 1 {
		writeReply(conn, controlwire.Reply{Error: "unauthorized"})
		return
	}
	log.Printf("control: %s %s %s", req.Method, req.Params.Game, req.Params.Profile)
	var res any
	var err error
	if req.Method == "hello" {
		res = controlwire.Hello{Version: version, Protocol: controlwire.Protocol}
	} else {
		res, err = h(ctx, req.Method, req.Params)
	}
	if err != nil {
		writeReply(conn, controlwire.Reply{Error: err.Error()})
		return
	}
	b, err := json.Marshal(res)
	if err != nil {
		writeReply(conn, controlwire.Reply{Error: err.Error()})
		return
	}
	writeReply(conn, controlwire.Reply{Result: b})
}

func writeReply(w io.Writer, rep controlwire.Reply) {
	b, err := json.Marshal(rep)
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}

// protocolChecked is set once the app has answered hello with a protocol this build speaks.
var protocolChecked atomic.Bool

// Call sends one request to the running app and decodes its result into out (nil to discard it). timeout bounds
// the whole call; methods that wait on the game take longer than reads.
func Call(method string, p Params, out any, timeout time.Duration) error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	if !protocolChecked.Load() {
		if err := controlwire.CheckProtocol(dir, timeout); err != nil {
			return err
		}
		protocolChecked.Store(true)
	}
	return controlwire.CallDir(dir, method, p, out, timeout)
}
