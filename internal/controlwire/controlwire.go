// Package controlwire is the client side of Mortar's local control channel: the discovery file, the wire shapes
// and one call. It imports nothing from the app, so the browser's native host can use it without the service graph.
package controlwire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

// FileName is the discovery file the app writes in its data folder while it runs.
const FileName = "control.json"

// Protocol is the control protocol this build speaks: the shape of requests, replies and control.json. The app
// states its own in control.json and in the hello reply, and a client refuses an app that speaks another.
const Protocol = 1

// MaxLine bounds a request or reply line.
const MaxLine = 16 << 20

// ErrNotRunning means no running Mortar answered.
var ErrNotRunning = errors.New("the Mortar app is not running: start it, then run this command again")

// Discovery is the content of FileName.
type Discovery struct {
	Port    int    `json:"port"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
	Version string `json:"version"`
	// Protocol is the app's control protocol.
	Protocol int `json:"protocol"`
}

// Hello is the reply to the hello method.
type Hello struct {
	Version  string `json:"version"`
	Protocol int    `json:"protocol"`
}

// Reply is one reply line.
type Reply struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Running reports whether a Mortar app with data folder dir is answering on its control channel.
func Running(dir string) bool {
	_, conn, err := dial(dir)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func dial(dir string) (Discovery, net.Conn, error) {
	var d Discovery
	root, err := os.OpenRoot(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil, ErrNotRunning
		}
		return d, nil, err
	}
	b, err := root.ReadFile(FileName)
	_ = root.Close()
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil, ErrNotRunning
		}
		return d, nil, err
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return d, nil, fmt.Errorf("control: %s: %w", FileName, err)
	}
	if d.Port < 1 || d.Port > 65535 {
		return d, nil, ErrNotRunning
	}
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(context.Background(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(d.Port)))
	if err != nil {
		return d, nil, ErrNotRunning
	}
	return d, conn, nil
}

// CallDir sends one request to the app whose data folder is dir and decodes its result into out (nil to discard
// it). timeout bounds the whole call.
func CallDir(dir, method string, params, out any, timeout time.Duration) error {
	d, conn, err := dial(dir)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	req, err := json.Marshal(map[string]any{"token": d.Token, "method": method, "params": params})
	if err != nil {
		return err
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return err
	}
	var rep Reply
	if err := json.NewDecoder(io.LimitReader(conn, MaxLine)).Decode(&rep); err != nil {
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

// CheckProtocol asks the app whose data folder is dir for its control protocol and refuses one this build does not
// speak.
func CheckProtocol(dir string, timeout time.Duration) error {
	var h Hello
	if err := CallDir(dir, "hello", nil, &h, timeout); err != nil {
		return err
	}
	if h.Protocol != Protocol {
		return fmt.Errorf("this mortar CLI speaks control protocol %d, the app speaks %d", Protocol, h.Protocol)
	}
	return nil
}
