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
}

// Reply is one reply line.
type Reply struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// CallDir sends one request to the app whose data folder is dir and decodes its result into out (nil to discard
// it). timeout bounds the whole call.
func CallDir(dir, method string, params, out any, timeout time.Duration) error {
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
	var d Discovery
	if err := json.Unmarshal(b, &d); err != nil {
		return fmt.Errorf("control: %s: %w", FileName, err)
	}
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(context.Background(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(d.Port)))
	if err != nil {
		return ErrNotRunning
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
