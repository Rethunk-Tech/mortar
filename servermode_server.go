//go:build server

package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// realDataOptIn lets a server build run on the account's own data folder.
const realDataOptIn = "MORTAR_SERVER_REAL_DATA"

// prepareServerMode stops a server build from running on the account's own data, which a desktop Mortar may be
// using at the same time: server mode has no single-instance lock. It also gives the server a free port when
// WAILS_SERVER_PORT is unset, where Wails would bind 8080, which another server build or any program may hold.
func prepareServerMode() error {
	if os.Getenv(realDataOptIn) != "1" && datadir.SharesAccountData() {
		return errors.New("server mode does not run on your own Mortar data; set HOME to a sandbox " +
			"(scripts/selftest.sh does), or set " + realDataOptIn + "=1 to run on it anyway")
	}
	if os.Getenv("WAILS_SERVER_PORT") != "" {
		return nil
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	if ok {
		_ = os.Setenv("WAILS_SERVER_PORT", strconv.Itoa(addr.Port))
	}
	return nil
}

// forwardLaunch hands a second launch's links and files to the Mortar already running on dataDir over the control
// channel and reports whether it did, since the single-instance hand-off a desktop build relies on is not compiled
// into a server build.
func forwardLaunch(dataDir string, args []string) bool {
	if len(args) == 0 {
		return false
	}
	if _, running := controlwire.Live(dataDir); !running {
		return false
	}
	for _, arg := range args {
		if !strings.Contains(arg, "://") && !strings.HasPrefix(arg, "-") {
			if abs, err := filepath.Abs(arg); err == nil {
				arg = abs
			}
		}
		if err := controlwire.CallDir(dataDir, control.OpenRequestMethod, map[string]string{"path": arg}, nil, 5*time.Second); err != nil {
			log.Printf("open request: %v", err)
		}
	}
	return true
}
