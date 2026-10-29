// Package control lets the mortar command line drive the running app. The app listens on 127.0.0.1 and writes the
// port and a random token to control.json in its data folder (readable by the user only); each connection carries
// one JSON request line and gets one JSON reply line. Only a process that can read that file can call it.
package control

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	// Version and File pick a release: a package or GitHub tag version, a Nexus file id or a GitHub asset name.
	Version string `json:"version,omitempty"`
	File    string `json:"file,omitempty"`
	// Body is a JSON payload a verb hands back that an earlier call produced (an undo token, a template).
	Body json.RawMessage `json:"body,omitempty"`
	// Preview asks a destructive verb to report what it would do instead.
	Preview bool `json:"preview,omitempty"`
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
	// Unscanned is "Install anyway" for an install the antivirus flagged.
	Unscanned bool   `json:"unscanned,omitempty"`
	Key       string `json:"key,omitempty"`
	Value     string `json:"value,omitempty"`
	Remove    bool   `json:"remove,omitempty"`
	Set       bool   `json:"set,omitempty"`
	Clear     bool   `json:"clear,omitempty"`
	Unlink    bool   `json:"unlink,omitempty"`
	Keep      int    `json:"keep,omitempty"`
	Preset    string `json:"preset,omitempty"`
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

// republishEvery is how often a running app checks that its discovery file is still there.
var republishEvery = 15 * time.Second

// answers reports whether an app answers on the data folder's control file.
func answers(ctx context.Context, dir string) bool {
	_, ok := controlwire.Live(ctx, dir)
	return ok
}

// staleTempAge is how old a leftover temp of the discovery file must be before it is removed: a younger one may be
// another running Mortar's write in progress.
const staleTempAge = time.Minute

// removeStaleTemps removes temp files earlier writes of path left behind. The app quitting while release writes the
// previous app's file back leaves one, since no write survives the process ending under it.
func removeStaleTemps(path string, now time.Time) {
	temps, _ := filepath.Glob(path + ".*.tmp")
	for _, temp := range temps {
		if info, err := os.Stat(temp); err == nil && now.Sub(info.ModTime()) > staleTempAge {
			_ = safefs.Remove(temp)
		}
	}
}

// republish writes the discovery file b back whenever it goes missing while the app runs, unless another app now
// answers on the folder.
func republish(ctx context.Context, every time.Duration, dir, path string, b []byte) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) && !answers(ctx, dir) {
				_ = safefs.WriteFileAtomic(path, b, 0o600)
			}
		}
	}
}

// release ends this app's claim on the discovery file b. Another instance may have taken the file over since, and
// removing its file would hide it from the CLI; the app this one replaced gets its own file back if it still answers.
func release(ctx context.Context, dir, path string, b, prev []byte) {
	cur, err := safefs.ReadFile(path)
	if err != nil || !bytes.Equal(cur, b) {
		return
	}
	if len(prev) > 0 && safefs.WriteFileAtomic(path, prev, 0o600) == nil && answers(ctx, dir) {
		return
	}
	_ = safefs.Remove(path)
}

// Serve listens until ctx ends, then removes the discovery file. It takes the file over from an app still running
// (a restart starts the new one before the old one is gone), and on exit gives the file back to that app if it still
// answers, so a short-lived second process on the data folder does not hide a long-running app from the CLI.
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
	removeStaleTemps(path, time.Now())
	prev, _ := safefs.ReadFile(path)
	b, err := json.Marshal(controlwire.Discovery{Port: tcp.Port, Token: token, PID: os.Getpid(), Version: version, Protocol: controlwire.Protocol})
	if err != nil {
		_ = ln.Close()
		return err
	}
	if err := safefs.WriteFileAtomic(path, b, 0o600); err != nil {
		_ = ln.Close()
		return err
	}
	go republish(ctx, republishEvery, dir, path, b)
	released := make(chan struct{})
	go func() {
		defer close(released)
		<-ctx.Done()
		_ = ln.Close()
		release(context.WithoutCancel(ctx), dir, path, b, prev)
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				<-released
				return ctx.Err()
			}
			return err
		}
		go serveConn(ctx, conn, token, version, h)
	}
}

const (
	// preAuthWait and preAuthBytes bound what a connection may take before it has shown the token.
	preAuthWait  = 10 * time.Second
	preAuthBytes = 64 << 10
)

// gateReader reads r until left bytes are used, then fails; the budget grows once the sender is known.
type gateReader struct {
	r    io.Reader
	left int64
}

func (g *gateReader) Read(p []byte) (int, error) {
	if g.left <= 0 {
		return 0, safefs.ErrTooLarge
	}
	if int64(len(p)) > g.left {
		p = p[:g.left]
	}
	n, err := g.r.Read(p)
	g.left -= int64(n)
	return n, err
}

var errUnauthorized = errors.New("unauthorized")

// readRequest decodes one request, checking the token as soon as it is read: until then the connection has wait to
// finish and a small body to do it in, and the token comes first on the wire from every client Mortar ships.
func readRequest(conn net.Conn, r io.Reader, token string, wait time.Duration) (request, error) {
	var req request
	_ = conn.SetReadDeadline(time.Now().Add(wait))
	gate := &gateReader{r: r, left: preAuthBytes}
	dec := json.NewDecoder(gate)
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return req, errors.New("control: not a request")
	}
	verified := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return req, err
		}
		switch keyTok {
		case "token":
			if err = dec.Decode(&req.Token); err != nil {
				return req, err
			}
			if subtle.ConstantTimeCompare([]byte(req.Token), []byte(token)) != 1 {
				return req, errUnauthorized
			}
			verified = true
			gate.left += controlwire.MaxLine
			_ = conn.SetReadDeadline(time.Time{})
		case "method":
			err = dec.Decode(&req.Method)
		case "params":
			err = dec.Decode(&req.Params)
		default:
			var skip json.RawMessage
			err = dec.Decode(&skip)
		}
		if err != nil {
			return req, err
		}
	}
	if !verified {
		return req, errUnauthorized
	}
	return req, nil
}

func serveConn(ctx context.Context, conn net.Conn, token, version string, h Handler) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReaderSize(conn, 64<<10)
	req, err := readRequest(conn, r, token, preAuthWait)
	if errors.Is(err, errUnauthorized) {
		writeReply(conn, controlwire.Reply{Error: "unauthorized"})
		return
	}
	if err != nil {
		return
	}
	log.Printf("control: %s %s %s", req.Method, req.Params.Game, req.Params.Profile)
	var res any
	if req.Method == "hello" {
		res = controlwire.Hello{Version: version, Protocol: controlwire.Protocol}
	} else {
		res, err = h(ctx, req.Method, req.Params)
	}
	if err != nil {
		rep := controlwire.Reply{Error: err.Error()}
		if k := usererr.KindOf(err); k != usererr.Unknown {
			rep.Kind = string(k)
		}
		if d := usererr.DetailOf(err); d != nil {
			rep.Detail, _ = json.Marshal(d)
		}
		writeReply(conn, rep)
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
