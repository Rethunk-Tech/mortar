//go:build server

package main

import (
	"context"
	"net"
	"os"
	"strconv"
)

// chooseServerPort gives server mode a free port when WAILS_SERVER_PORT is unset, where Wails would bind 8080, which
// another server build or any program may hold. Wails logs the address it listens on.
func chooseServerPort() {
	if os.Getenv("WAILS_SERVER_PORT") != "" {
		return
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	if ok {
		_ = os.Setenv("WAILS_SERVER_PORT", strconv.Itoa(addr.Port))
	}
}
