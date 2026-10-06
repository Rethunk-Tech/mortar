// Package bridge talks to a loader's companion mod: a small mod inside the running game that listens on loopback and
// publishes its port and token in a state file. Mortar sends it console commands or asks it questions; the one-line
// reply is `ok`, `ok <json>` or `error: <message>`.
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
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Companion is a loader's companion mod: its folder and id as the loader spells them, and the state file it writes.
// StateFile is relative to the mod's folder for SMAPI and to the profile for BepInEx, which keeps one config folder.
type Companion struct {
	ModFolder, ID, StateFile string
}

// SMAPI is the Mortar SMAPI Bridge.
var SMAPI = Companion{ModFolder: "MortarSmapiBridge", ID: "Rethunk.MortarSmapiBridge", StateFile: "mortar-smapi-bridge.json"}

// BepInEx is the Mortar BepInEx Bridge, whose state file sits in BepInEx's config folder.
var BepInEx = Companion{ModFolder: "MortarBepInExBridge", ID: "Rethunk.MortarBepInExBridge", StateFile: "BepInEx/config/mortar-bepinex-bridge.json"}

// maxReply bounds the reply read: a status answer is one line of JSON.
const maxReply = 1 << 20

// timeout bounds connecting and each read or write; a variable so tests can shorten it.
var timeout = 3 * time.Second

var (
	// ErrNotRunning means nothing answers on the port the state file names: a game that ended without removing it.
	ErrNotRunning = errors.New("the game is not running")
	// ErrNotReady means the game is up but the bridge has not started yet, or the mod is missing.
	ErrNotReady = errors.New("the bridge is not ready yet: wait until the loader has finished loading")
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
}

// ReadState reads the companion's state file. A missing or unreadable file is ErrNotReady. Whether the game behind it
// still runs only a connection can tell: the pid a companion records is the game's own, a Wine PID under Proton.
func ReadState(stateFile string) (State, error) {
	b, err := fsx.ReadFile(stateFile)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, ErrNotReady
	}
	if err != nil {
		return State{}, err
	}
	var st State
	if json.Unmarshal(b, &st) != nil || st.Port <= 0 || st.Port > 65535 || st.Token == "" {
		return State{}, ErrNotReady
	}
	return st, nil
}

// Send runs command in the game whose companion wrote stateFile.
func Send(ctx context.Context, stateFile, command string) error {
	st, err := ReadState(stateFile)
	if err != nil {
		return err
	}
	_, err = send(ctx, st, command)
	return err
}

// Query asks the game a question and returns the JSON the companion answers with. what is status, plugins, perf or
// perf start.
func Query(ctx context.Context, stateFile, what string) (json.RawMessage, error) {
	if what != "status" && what != "plugins" && what != "perf" && what != "perf start" {
		return nil, fmt.Errorf("unknown query %q", what)
	}
	st, err := ReadState(stateFile)
	if err != nil {
		return nil, err
	}
	payload, err := send(ctx, st, what)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(payload)) {
		return nil, fmt.Errorf("unexpected reply from the bridge: %q", payload)
	}
	return json.RawMessage(payload), nil
}

// send returns the JSON after `ok`, "" for a bare `ok`.
func send(ctx context.Context, st State, command string) (string, error) {
	if strings.ContainsAny(command, "\r\n") {
		return "", &RejectedError{Message: "a command is one line"}
	}
	dialer := net.Dialer{Timeout: timeout}
	// Mortar starts the game in the host's network namespace, so the companion's loopback is this one.
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(st.Port)))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotRunning, err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	if _, err := io.WriteString(conn, st.Token+"\n"+command+"\n"); err != nil {
		return "", err
	}
	reply, err := bufio.NewReader(io.LimitReader(conn, maxReply)).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("no reply from the bridge: %w", err)
	}
	reply = strings.TrimSpace(reply)
	switch {
	case reply == "ok":
		return "", nil
	case strings.HasPrefix(reply, "ok "):
		return strings.TrimPrefix(reply, "ok "), nil
	case reply == "error: unauthorized":
		return "", ErrUnauthorized
	case strings.HasPrefix(reply, "error: "):
		return "", &RejectedError{Message: strings.TrimPrefix(reply, "error: ")}
	}
	return "", fmt.Errorf("unexpected reply from the bridge: %q", reply)
}
