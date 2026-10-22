package main

import (
	"bytes"
	"log/slog"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// applyGraphics sets the WebKit rendering the Graphics setting asks for. WebKit reads these variables when the window's
// web view starts, after the settings open. A variable set in the environment Mortar was started with wins.
func applyGraphics(mode string) {
	started := startEnv()
	set := func(name, value string) {
		if _, ok := started[name]; !ok {
			_ = os.Setenv(name, value)
		}
	}
	switch mode {
	case settings.GraphicsSmooth:
		// "0" keeps the DMA-BUF renderer that Wails turns off where NVIDIA renders.
		set("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
		set("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	case settings.GraphicsCompatible:
		set("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	slog.Info("graphics", "mode", mode, "WEBKIT_DISABLE_DMABUF_RENDERER", os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"),
		"WEBKIT_DISABLE_COMPOSITING_MODE", os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE"))
}

// startEnv is the environment the process was started with, before Wails' init changed it.
func startEnv() map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile("/proc/self/environ")
	if err != nil {
		return out
	}
	for kv := range bytes.SplitSeq(raw, []byte{0}) {
		if k, v, ok := bytes.Cut(kv, []byte{'='}); ok {
			out[string(k)] = string(v)
		}
	}
	return out
}
