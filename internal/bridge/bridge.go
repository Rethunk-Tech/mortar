// Package bridge sends console commands to a running game through the Mortar SMAPI Bridge mod, which listens
// on loopback and publishes its port and token in a state file in its own folder.
package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

const (
	// ModFolder is the bridge mod's folder inside its store entry.
	ModFolder = "MortarSmapiBridge"
	// UniqueID is the bridge mod's manifest UniqueID.
	UniqueID = "Rethunk.MortarSmapiBridge"
	// StateFile is written by the mod on start and removed on exit.
	StateFile = "mortar-smapi-bridge.json"

	// maxReply bounds the reply read: the mod answers with one short line.
	maxReply = 4096
)

// timeout bounds connecting and each read or write; a variable so tests can shorten it.
var timeout = 3 * time.Second

var (
	// ErrNotRunning means the game process the state file names is gone.
	ErrNotRunning = errors.New("the game is not running")
	// ErrNotReady means the game is up but the bridge has not started yet, or the mod is missing.
	ErrNotReady = errors.New("the bridge is not ready yet: wait until SMAPI has finished loading")
	// ErrUnauthorized means the bridge did not accept the token.
	ErrUnauthorized = errors.New("the bridge refused the token")
)

// RejectedError is a command the bridge refused, with its reason.
type RejectedError struct{ Message string }

func (e *RejectedError) Error() string { return "command rejected: " + e.Message }

// State is the content of the state file.
type State struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

// ReadState reads the state file in the bridge mod's folder dir. A missing or unreadable file is ErrNotReady,
// a file whose process has exited ErrNotRunning.
func ReadState(dir string) (State, error) {
	b, err := fsx.ReadFile(filepath.Join(dir, StateFile))
	if errors.Is(err, os.ErrNotExist) {
		return State{}, ErrNotReady
	}
	if err != nil {
		return State{}, err
	}
	var st State
	if json.Unmarshal(b, &st) != nil || st.Port <= 0 || st.Port > 65535 || st.Token == "" || st.PID <= 0 {
		return State{}, ErrNotReady
	}
	if !launch.Alive(st.PID) {
		return State{}, ErrNotRunning
	}
	return st, nil
}

// Send runs command in the game whose bridge mod lives in dir.
func Send(dir, command string) error {
	st, err := ReadState(dir)
	if err != nil {
		return err
	}
	return send(st, command)
}

func send(st State, command string) error {
	if strings.ContainsAny(command, "\r\n") {
		return &RejectedError{Message: "a command is one line"}
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(context.Background(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(st.Port)))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotReady, err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, st.Token+"\n"+command+"\n"); err != nil {
		return err
	}
	reply, err := bufio.NewReader(io.LimitReader(conn, maxReply)).ReadString('\n')
	if err != nil {
		return fmt.Errorf("no reply from the bridge: %w", err)
	}
	switch reply = strings.TrimSpace(reply); {
	case reply == "ok":
		return nil
	case reply == "error: unauthorized":
		return ErrUnauthorized
	case strings.HasPrefix(reply, "error: "):
		return &RejectedError{Message: strings.TrimPrefix(reply, "error: ")}
	}
	return fmt.Errorf("unexpected reply from the bridge: %q", reply)
}
