//go:build server

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"

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
